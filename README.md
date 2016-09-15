# Objdbstore 

A session store backend for [gorilla/sessions](http://www.gorillatoolkit.org/pkg/sessions) - [src](https://github.com/gorilla/sessions) using either [Etcd](https://coreos.com/etcd/) or [Consul](https://www.consul.io/).

## Requirements

Depends on contiv objdb [contiv-objdb](https://github.com/contiv/objdb).

## Installation

    go get github.com/shampur/objdbstore

## Documentation

Available on [godoc.org](http://www.godoc.org/github.com/shampur/objdbstore).

See http://www.gorillatoolkit.org/pkg/sessions for full documentation on underlying interface.

### Example

    // Fetch new store.
	addrs := []string{"127.0.0.1:2379"}
	store := NewObjdbStore([]string{"http://127.0.0.1:2379"}, "session-name", "etcd",[]byte("something-very-secret")),

    // Get a session.
	session, err := store.Get(req, "session-key")
	if err != nil {
        log.Error(err.Error())
    }

    // Add a value.
    session.Values["foo"] = "bar"

    // Save.
    if err = session.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

    // Delete session.
    session.Options.MaxAge = -1
    if err = session.Save(req, rsp); err != nil {
        t.Fatalf("Error saving session: %v", err)
    }

