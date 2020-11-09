package memcached

import "context"

type RequestHandler interface{
    Stats(*Stats)
}

// A Getter is an object who responds to a simple
// "get" command.
type Getter interface {
	RequestHandler
	Get(string) MemcachedResponse
	GetWithContext(*context.Context, string) MemcachedResponse
}

// A Setter is an object who response to a simple
// "set" command.
type Setter interface {
	RequestHandler
	Set(*Item) MemcachedResponse
	SetWithContext(*context.Context, *Item) MemcachedResponse
}

// A Deleter is an object who responds to a simple
// "delete" command.
type Deleter interface {
	RequestHandler
	Delete(string) MemcachedResponse
	DeleteWithContext(*context.Context, string) MemcachedResponse
}
