package database

import (
	"github.com/globalsign/mgo/bson"
	"github.com/markbates/goth"
	"github.com/blizztrack/owmods/core"
	"log"
	"strings"
)

type User struct {
	ID        bson.ObjectId `bson:"_id,omitempty" json:"-"`
	NickName  string        `bson:"nick"`
	BattleID  string        `bson:"bid"`
	URL       string        `bson:"url" form:"url"`
	ShortID   string        `bson:"short_id"`
	Name      string        `bson:"name" form:"name"`
	NameLower string        `bson:"name_lower"`
	Image     string        `bson:"image" form:"-"`
	ImagePath string        `bson:"image_path" form:"-"`

	// NoBSON
	Mode string `bson:"-" form:"mode"`
	Type string `bson:"-" form:"type"`
	Code string `bson:"-" form:"code"` // Never used
}

const userTable = "users"

func UpsertUser(current goth.User) User {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(userTable)
	defer session.Close()

	user := User{
		NickName: current.NickName,
		BattleID: current.UserID,
	}

	cU := GetUser(user.BattleID)
	if cU.BattleID == current.UserID {
		cU.NickName = user.NickName

		if cU.ShortID == "" {
			cU.ShortID = core.ShortID().MustGenerate()
		}

		err = c.Update(bson.M{"bid": cU.BattleID}, cU)
		if err != nil {
			log.Printf("UpdateUser Failed: %v", err)
		}
	} else {
		user.ShortID = core.ShortID().MustGenerate()
		err = c.Insert(user)
		if err != nil {
			log.Printf("UpdateUser Failed: %v", err)
		}
	}

	return GetUser(user.BattleID)
}

func UpdateUser(user User) User {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(userTable)
	defer session.Close()

	err = c.Update(bson.M{"bid": user.BattleID}, user)
	if err != nil {
		log.Printf("UpdateUser Failed: %v", err)
	}

	return user
}

func GetUser(id string) User {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(userTable)
	defer session.Close()

	var user User
	if !bson.IsObjectIdHex(id) {
		err = c.Find(bson.M{"$or": []bson.M{
			{"bid": id},
			{"url": &bson.RegEx{
				Pattern: id,
				Options: "i",
			}},
			{"short_id": id},
		}}).One(&user)
		if err != nil {
			log.Printf("GetUser Failed bson.$OR: %v", err)
			return User{}
		}
	} else {
		err = c.Find(bson.M{"_id": bson.ObjectIdHex(id)}).One(&user)
		if err != nil {
			log.Printf("GetUser Failed bson.ID: %v", err)
			return User{}
		}
	}

	return user
}

func BulkFindUser(ids ...string) []User {
	db, err := core.Instance()
	if err != nil {
		log.Panic(err)
	}

	session, c := db.Collection(userTable)
	defer session.Close()

	var user []User
	c.Find(bson.M{"bid": bson.M{"$in": ids}}).All(&user)

	return user
}

func (u User) GetNickNameNoTag() string {
	return strings.Split(u.NickName, "#")[0]
}

func (u User) GetID() string {
	if u.ShortID == "" {
		return u.ID.Hex()
	}

	return u.ShortID
}
