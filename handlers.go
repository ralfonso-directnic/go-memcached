package memcached

import "context"

type RequestHandler interface{

}

type StatsHandler interface {
	RequestHandler
	Stats(Stats)
}

// A Getter is an object who responds to a simple
// "get" command.
type Getter interface {
	RequestHandler
	GetWithContext(*context.Context, string) MemcachedResponse
}

// A Setter is an object who response to a simple
// "set" command.
type Setter interface {
	RequestHandler
	SetWithContext(*context.Context, *Item) MemcachedResponse
}

// A Deleter is an object who responds to a simple
// "delete" command.
type Deleter interface {
	RequestHandler
	DeleteWithContext(*context.Context, string) MemcachedResponse
}
