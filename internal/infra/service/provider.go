package service

import (
	"github.com/bool64/brick-starter-kit/internal/domain/greeting"
)

// GreetingMakerProvider is a service provider.
type GreetingMakerProvider interface {
	GreetingMaker() greeting.Maker
}

// GreetingClearerProvider is a service provider.
type GreetingClearerProvider interface {
	GreetingClearer() greeting.Clearer
}
