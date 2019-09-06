package core

import (
	"github.com/teris-io/shortid"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	sId *shortid.Shortid

	shortIDSeed = kingpin.Flag("short_id_seed", "Short ID Seed").Envar("SHORT_ID_SEED").Default("1").Uint64()
)

func ShortID() *shortid.Shortid {
	if sId == nil {
		sId = shortid.MustNew(1, shortid.DefaultABC, *shortIDSeed)
	}

	return sId
}
