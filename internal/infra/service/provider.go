package service

import (
	"github.com/vearutop/photo-blog/internal/domain/greeting"
)

// GreetingMakerProvider is a service provider.
type GreetingMakerProvider interface {
	GreetingMaker() greeting.Maker
}

// GreetingClearerProvider is a service provider.
type GreetingClearerProvider interface {
	GreetingClearer() greeting.Clearer
}
