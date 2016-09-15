package objdbstore

import (
	"errors"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"sync"
	"github.com/contiv/objdb"
	"net/http"
	"strings"
	"encoding/base32"
	"encoding/json"
)

var ErrNoDatabase = errors.New("no databases available")

// Amount of time for cookies to expire.
var sessionExpire = 86400 * 30

type ObjdbStore struct {
	Clientapi     	objdb.API          	// objdb client api reference
	Bucket        	string               	// bucket to store sessions in
	Codecs        	[]securecookie.Codec 	// session codecs
	Options       	*sessions.Options    	// default configuration
	DefaultMaxAge 	int 			// default TTL for a MaxAge == 0 session
	StoreMutex 	sync.RWMutex
	cache 		map[string]string
}

type sessionObj struct {
	Value	string `json:"value"`
}


// NewObjdbStore returns a new objdb store
func NewObjdbStore(endpoints []string, bucket string, pluginclient string,keyPairs ...[]byte) *ObjdbStore {
	return &ObjdbStore{
		Clientapi: func() objdb.API {
				pluginStore := objdb.GetPlugin(pluginclient)
				api, err := pluginStore.NewClient(endpoints)
				if err != nil {
					panic("plugin store not found ")
				}
				return api
				}(),
		Bucket: bucket,
		Codecs: securecookie.CodecsFromPairs(keyPairs...),
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: sessionExpire,
		},
	}
}

// Get returns a session for the given name after adding it to the registry.
func (s *ObjdbStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(s, name)
}

// New returns a session for the given name without adding it to the registry.
func (s *ObjdbStore) New(r *http.Request, name string) (*sessions.Session, error) {
	var err error
	session := sessions.NewSession(s, name)
	opts := *s.Options
	session.Options = &opts
	session.IsNew = true
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, s.Codecs...)
		if err == nil {
			err := s.load(session)
			session.IsNew = !(err == nil) // err == nil if session key already present in objectdb
		}
	}
	return session, err
}

// Save adds a single session to the response.
func (s *ObjdbStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	// Marked for deletion.
	if session.Options.MaxAge < 0 {
		if err := s.delete(session); err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
	} else {
		// Build an alphanumeric key for the objdb store
		if session.ID == "" {
			session.ID = strings.TrimRight(base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
		}
		if err := s.save(session); err != nil {
			return err
		}
		encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, s.Codecs...)
		if err != nil {
			return err
		}
		http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	}
	return nil
}

// save stores the session to object store
func (s *ObjdbStore) save(session *sessions.Session) error {

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, s.Codecs...)
	if err != nil {
		return err
	}
	var sobj sessionObj
	sobj.Value = encoded

	bytedata, err := json.Marshal(sobj)

	if err!= nil {
		return err
	}
	s.StoreMutex.Lock()
	key := "session_" + session.ID
	err = s.Clientapi.SetObj(key, bytedata)
	s.StoreMutex.Unlock()

	if err!=nil {
		return err
	}

	return err
}

// load reads the session from object store
func (s *ObjdbStore) load(session *sessions.Session) error {
	//Check if the session details are present in cache

	var  sval sessionObj
	s.StoreMutex.Lock()
	key := "session_" + session.ID
	err := s.Clientapi.GetObj(key, &sval)
	s.StoreMutex.Unlock()
	if err != nil {
		return err
	}

	if err = securecookie.DecodeMulti(session.Name(), sval.Value,
		&session.Values, s.Codecs...); err != nil {
		return err
	}

	return nil
}

func (s *ObjdbStore) delete(session *sessions.Session) error {
	s.StoreMutex.Lock()
	key := "session_" + session.ID
	err := s.Clientapi.DelObj(key)
	s.StoreMutex.Unlock()
	return err
}
