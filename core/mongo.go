package core

import (
	"errors"
	"log"

	"github.com/globalsign/mgo"
)

type MongoSettings struct {
	Host, Username, Password, Database string
}

type Mongo struct {
	session  *mgo.Session
	settings MongoSettings
}

var instance *Mongo

func NewMongo(settings MongoSettings) *Mongo {
	session, err := mgo.Dial(settings.Host)
	if err != nil {
		// This should never happen but if it does we need to panic... this can cause some wonky effects if we don't...
		log.Panicln(err)
	}
	session.SetMode(mgo.Monotonic, true)

	if len(settings.Username) > 0 && len(settings.Password) > 0 {
		err := session.Login(&mgo.Credential{
			Username: settings.Username,
			Password: settings.Password,
		})

		if err != nil {
			// Panic when we failed to login because well... go build in logger has no warning...
			// Maybe i should replace the built in logger later... Iris has one built in that we could make public
			log.Panicln(err)
		}
	}

	instance = &Mongo{
		session:  session,
		settings: settings,
	}

	return instance
}

func (mg *Mongo) copySession() *mgo.Session {
	return mg.session.Copy()
}

func (mg *Mongo) Collection(collection string) (*mgo.Session, *mgo.Collection) {
	session := mg.copySession()
	c := session.DB(mg.settings.Database).C(collection)

	return session, c
}

func Instance() (*Mongo, error) {
	if instance == nil {
		return nil, errors.New("mongo not configured please run new()")
	}
	return instance, nil
}