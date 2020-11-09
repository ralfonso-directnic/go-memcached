# Memcached library for Go

## What even is this?
Modeled similarly to the stdlib `net/http` package, `memcached` gives you a simple interface to building your own memcached protocol compatible applications.

## Install
```
$ go get github.com/ralfonso-directnic/go-memcached
```

## Interfaces
Implement as little or as much as you'd like.
```go
type Getter interface {
	RequestHandler
	GetWithContext(*context.Context, string) MemcachedResponse
}

type Setter interface {
	RequestHandler
	SetWithContext(*context.Context, *Item) MemcachedResponse
}

type Deleter interface {
	RequestHandler
	DeleteWithContext(*context.Context, string) MemcachedResponse
}
```

## Hello World
```go
package main

import (
	memcached "github.com/ralfonso-directnic/go-memcached"
)

type Cache struct {
    
    
}

// handle stats in your getter/setters
func (c *Cache) Stats(s *memcached.Stats){
    
}   

func (c *Cache) GetWithContext(ctx *context.Context, key string) memcached.MemcachedResponse {
	if key == "hello" {
		item = &memcached.Item{
			Key: key,
			Value: []byte("world"),
		}
		return item, nil
	}
	return nil, memcached.NotFound
}

func main() {
	server := memcached.NewServer(":11211", &Cache{})
	server.ListenAndServe()
}
```

## Examples
 * [Simple Memcached](examples/memcached.go)  *Don't actually use this*

## Documentation
 * [http://godoc.org/github.com/ralfonso-directnic/go-memcached](http://godoc.org/github.com/mattrobenolt/go-memcached)
