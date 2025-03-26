package qlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bool64/ctxd"
	"github.com/bool64/sqluct"
)

const (
	messageTable = "message"
	archiveTable = "message_archive"
)

type UnixTime int64

type message struct {
	ID          int                            `db:"id,omitempty" json:"id,omitempty"`
	CreatedAt   UnixTime                       `db:"created_at" json:"created_at,omitzero" title:"Created at"`
	TryAfter    UnixTime                       `db:"try_after" json:"try_after,omitempty" title:"Try after"`
	StartedAt   UnixTime                       `db:"started_at" json:"started_at,omitempty" title:"Started at" description:"Consumption start, can not be picked again"`
	ProcessedAt UnixTime                       `db:"processed_at" json:"processed_at,omitempty" title:"Processed at" description:"Processed message does not need to be consumed."`
	Header      sqluct.JSON[map[string]string] `db:"header" json:"header,omitzero"`
	Elapsed     float64                        `db:"elapsed" json:"elapsed,omitempty" title:"Elapsed time in seconds"`
	Topic       string                         `db:"topic" json:"topic" title:"Topic" required:"true"`
	Error       string                         `db:"error" json:"error,omitempty" title:"Error message"`
	Tries       int                            `db:"tries" json:"tries,omitempty" title:"Tries count"`
	OnSuccess   sqluct.JSON[[]Message]         `db:"on_success" json:"on_success,omitzero" title:"Publish these messages after successful processing"`
}

type Message struct {
	message
	Payload sqluct.JSON[any] `db:"payload" json:"payload,omitempty" title:"Val"`
}

type MessageOf[V any] struct {
	message
	Payload sqluct.JSON[V] `db:"payload" json:"payload,omitempty" title:"Val"`
}

func MakeMessage(topic string, payloadValue any) Message {
	m := Message{}
	m.Topic = topic
	m.Payload.Val = payloadValue

	return m
}

func (m *Message) PublishOnSuccess(topic string, payloadValue any) {
	m.OnSuccess.Val = append(m.OnSuccess.Val, MakeMessage(topic, payloadValue))
}

type ConsumerOptions struct {
	Concurrency int
	StartExpire time.Duration
}

type consumerOf[V any] struct {
	consume     func(ctx context.Context, v V) error
	sem         chan struct{}
	startExpire int
	logger      ctxd.Logger
}

func (c consumerOf[V]) assertType(payload any, topic string) error {
	var v V

	if _, ok := payload.(V); ok {
		return nil
	}

	return fmt.Errorf("invalid payload for topic %s, expected %T, received %T", topic, v, payload)
}

func (c consumerOf[V]) pollOnce(b *Broker, topic string) error {
	free := cap(c.sem) - len(c.sem)
	if free == 0 {
		return nil
	}

	q := b.st.SelectStmt(messageTable, MessageOf[V]{}).
		Where(b.ref.Fmt(
			"%s = ? AND %s < unixepoch() AND %s < unixepoch() - ? AND %s = 0",
			&b.r.Topic, &b.r.TryAfter, &b.r.StartedAt, &b.r.ProcessedAt), topic, c.startExpire,
		).
		OrderByClause(b.ref.Fmt("%s ASC", &b.r.ID)).
		Limit(uint64(free))

	var msgs []MessageOf[V]

	if err := b.st.Select(context.Background(), q, &msgs); err != nil {
		return err
	}

	c.logger.Debug(context.Background(), "messages found", "messages", msgs)

	found := false

	for _, msg := range msgs {
		ctx := context.Background()

		msg.StartedAt = UnixTime(time.Now().Unix())

		res, err := b.st.UpdateStmt(messageTable, nil).
			Set(b.ref.Col(&b.r.StartedAt), msg.StartedAt).
			Where(b.ref.Fmt("%s = ?", &b.r.ID), msg.ID).
			Exec()
		if err != nil {
			return err
		}

		aff, err := res.RowsAffected()
		if err != nil {
			return err
		}

		if aff > 0 {
			found = true

			c.sem <- struct{}{}
			go c.consumeOnce(ctx, b, msg)
		}
	}

	if found {
		select {
		case b.pollAgain <- true:
		default:
		}
	}

	return nil
}

func (c consumerOf[V]) consumeOnce(ctx context.Context, b *Broker, msg MessageOf[V]) {
	defer func() {
		<-c.sem
	}()

	c.logger.Debug(ctx, "consumeOnce", "message", msg)

	start := time.Now()
	err := c.consume(ctx, msg.Payload.Val)
	if err == nil {
		msg.ProcessedAt = UnixTime(time.Now().Unix())

		for _, cb := range msg.OnSuccess.Val {
			if err := b.Publish(ctx, cb.Topic, cb.Payload.Val, func(msg *Message) {
				*msg = cb
			}); err != nil {
				b.logError(ctx, err)
			}
		}
	} else {
		msg.Error = err.Error()
	}

	msg.Elapsed += time.Since(start).Seconds()
	msg.Tries++

	var er ErrRetryAfter
	if errors.As(err, &er) {
		msg.StartedAt = 0
		msg.TryAfter = UnixTime(time.Time(er).Unix())
		msg.ProcessedAt = 0
	} else {
		msg.ProcessedAt = UnixTime(time.Now().Unix())
	}

	if msg.ProcessedAt > 0 {
		id := msg.ID
		msg.ID = 0

		if _, err := b.st.InsertStmt(archiveTable, msg).ExecContext(ctx); err != nil {
			b.logError(ctx, err)
			return
		}

		if _, err := b.st.DeleteStmt(messageTable).
			Where(b.ref.Fmt("%s = ?", &b.r.ID), id).ExecContext(ctx); err != nil {
			b.logError(ctx, err)
		}

		return
	}

	c.logger.Debug(ctx, "updating message", "message", msg, "error", err)

	_, err = b.st.UpdateStmt(messageTable, msg).Where(b.ref.Fmt("%s = ?", &b.r.ID), msg.ID).ExecContext(ctx)
	if err != nil {
		b.logError(ctx, err)
	}
}

func AddConsumer[V any](b *Broker, topic string, consume func(ctx context.Context, v V) error, options ...func(o *ConsumerOptions)) error {
	if _, ok := b.pollTopic[topic]; ok {
		return fmt.Errorf("consumer for topic %s already exists", topic)
	}

	opts := ConsumerOptions{
		Concurrency: 1,
		StartExpire: time.Hour,
	}

	for _, o := range options {
		o(&opts)
	}

	if opts.Concurrency == 0 {
		return fmt.Errorf("concurrency for topic %s is zero", topic)
	}

	c := consumerOf[V]{
		consume:     consume,
		sem:         make(chan struct{}, opts.Concurrency),
		startExpire: int(opts.StartExpire.Seconds()),
		logger:      b.Logger,
	}

	b.pollTopic[topic] = c.pollOnce
	b.assertType[topic] = c.assertType

	return nil
}

type Broker struct {
	Logger ctxd.Logger

	st  *sqluct.Storage
	r   *Message
	ref *sqluct.Referencer

	pollTopic  map[string]func(b *Broker, topic string) error
	assertType map[string]func(payload any, topic string) error
	pollAgain  chan bool
}

type ErrRetryAfter time.Time

func (e ErrRetryAfter) Error() string {
	return "retry after: " + string(time.Time(e).Format(time.RFC3339))
}

func NewBroker(storage *sqluct.Storage) *Broker {
	b := &Broker{
		Logger:     ctxd.NoOpLogger{},
		st:         storage,
		r:          &Message{},
		ref:        storage.MakeReferencer(),
		pollTopic:  make(map[string]func(b *Broker, topic string) error),
		assertType: make(map[string]func(payload any, topic string) error),
		pollAgain:  make(chan bool, 2),
	}

	b.ref.AddTableAlias(b.r, messageTable)

	go b.poll()

	return b
}

func (b *Broker) validate(msg Message) error {
	assertType := b.assertType[msg.Topic]
	if assertType == nil {
		return fmt.Errorf("no consumer for topic: %s", msg.Topic)
	}

	if err := assertType(msg.Payload.Val, msg.Topic); err != nil {
		return err
	}

	for _, cb := range msg.OnSuccess.Val {
		if err := b.validate(cb); err != nil {
			return err
		}
	}

	return nil
}

func (b *Broker) Publish(ctx context.Context, topic string, payloadValue any, options ...func(msg *Message)) error {
	msg := Message{}

	for _, opt := range options {
		opt(&msg)
	}

	msg.Topic = topic
	msg.Payload.Val = payloadValue
	msg.CreatedAt = UnixTime(time.Now().Unix())

	if err := b.validate(msg); err != nil {
		return err
	}

	b.Logger.Debug(ctx, "publishing message", "msg", msg)

	res, err := b.st.InsertStmt(messageTable, msg).ExecContext(ctx)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	msg.ID = int(id)

	select {
	case b.pollAgain <- true:
		b.Logger.Debug(ctx, "polling after publish", "msg", msg)
	default:
		b.Logger.Debug(ctx, "already polling after publish", "msg", msg)
	}

	return nil
}

func (b *Broker) poll() {
	for <-b.pollAgain {
		b.Logger.Debug(context.Background(), "poll again")
		if err := b.pollOnce(); err != nil {
			b.logError(context.Background(), err)
		}
	}
}

func (b *Broker) pollOnce() error {
	for topic, poll := range b.pollTopic {
		b.Logger.Debug(context.Background(), "poll once", "topic", topic)
		if err := poll(b, topic); err != nil {
			return err
		}
	}

	return nil
}

func (b *Broker) Poll() {
	select {
	case b.pollAgain <- true:
		b.Logger.Debug(context.Background(), "polling queues")
	default:
		b.Logger.Debug(context.Background(), "already polling queue")
	}
}

func (b *Broker) Close() {
	close(b.pollAgain)
}

func (b *Broker) logError(ctx context.Context, err error) {
	b.Logger.Error(ctx, "qlite failed", "error", err)
}
