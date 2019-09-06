package main

import (
	"github.com/blizztrack/owmods/core"
	"github.com/blizztrack/owmods/core/paging"
	"github.com/blizztrack/owmods/database"
	"github.com/blizztrack/owmods/system"
	"github.com/blizztrack/owmods/tools"
	"github.com/blizztrack/owmods/web_server/helpers"
	"github.com/blizztrack/owmods/web_server/routes"
	"github.com/globalsign/mgo/bson"
	"github.com/kataras/iris"
	"github.com/kataras/iris/middleware/logger"
	"github.com/kataras/iris/middleware/recover"
	"gopkg.in/alecthomas/kingpin.v2"
	"path"
	"strconv"
)

var (
	listen = kingpin.Flag("listen", "Listening port").Short('l').Envar("LISTEN").Default(":1337").String()
	assets = kingpin.Flag("assets", "Assets folder").Short('a').Envar("ASSETS").Default("./assets").String()

	/* Mongo DB */
	mongoHost = kingpin.Flag("mongo_host", "Mongo Host").Envar("MONGO_HOST").Default("localhost:27017").String()
	mongoUser = kingpin.Flag("mongo_user", "Mongo User").Envar("MONGO_USER").Default("").String()
	mongoPass = kingpin.Flag("mongo_pass", "Mongo Pass").Envar("MONGO_PASS").Default("").String()
	mongoDB   = kingpin.Flag("mongo_db", "Mongo Pass").Envar("MONGO_DB").Default("owmods").String()
)

const Description = "A simple website that will let you share your workshop creations!"

func main() {
	kingpin.Parse()
	core.NewMongo(core.MongoSettings{
		Host:     *mongoHost,
		Username: *mongoUser,
		Password: *mongoPass,
		Database: *mongoDB,
	})
	core.NewRedis()

	// Need to verify this thing
	tools.ImportOldGames()

	app := iris.New()
	app.Use(recover.New())
	app.Use(logger.New())

	iris.RegisterOnInterrupt(func() {
		system.Redis().Client().Close()
	})
	defer system.Redis().Client().Close() // close the database connection if application errored.

	app.Use(func(ctx iris.Context) {
		ctx.ViewData("Header", core.Render.Header(
			"OWMods",
			"bt",
			Description,
		))

		ctx.ViewData("A_Page", "")

		sess := system.Session().Get(ctx)
		val := sess.Get("User")
		if val == nil {
			ctx.ViewData("User", false)
		} else {
			/*
				ctx.ViewData("User", val)
				ctx.ViewData("UserID", sess.Get("UserID"))
			*/

			ctx.ViewData("User", database.GetUser(sess.Get("UserID").(string)))
		}

		ctx.Gzip(true)
		ctx.Next()
	})

	tmpEngine := core.NewRender(path.Join(*assets, "views"), ".tmpl").Defaults()
	app.RegisterView(tmpEngine.ViewEngine)

	app.StaticWeb("/static", path.Join(*assets, "static"))

	app.Get("/ads.txt", func(ctx iris.Context) {
		ctx.ServeFile(path.Join(*assets, "static", "ads.txt"), true)
	})
	app.Get("/robots.txt", func(ctx iris.Context) {
		ctx.ServeFile(path.Join(*assets, "static", "robots.txt"), true)
	})
	app.Get("/favicon.ico", func(ctx iris.Context) {
		ctx.ServeFile(path.Join(*assets, "static", "favicon.ico"), true)
	})


	app.Get("/", func(ctx iris.Context) {
		page := ctx.URLParamIntDefault("page", 1)
		query := ctx.URLParamDefault("search", "")
		sort := ctx.URLParamDefault("sort", "1")
		switch sort {
		case "1":
			sort = "-posted"
			break
		case "2":
			sort = "+posted"
			break
		case "3":
			sort = "-updated"
			break
		default:
			sort = "-posted"
			break
		}

		urlQuery := ctx.URLParams()
		if _, ok := urlQuery["page"]; !ok {
			urlQuery["page"] = strconv.Itoa(page)
		}

		searchQuery := bson.M{}
		searchQuery["privacy"] = bson.M{
			"$in": []interface{}{nil, 0},
		}

		if query != "" {
			searchQuery["$or"] = []bson.M{
				{"title": &bson.RegEx{
					Pattern: query,
					Options: "i",
				},},
				{"tldr": &bson.RegEx{
					Pattern: query,
					Options: "i",
				},},
			}
		}

		workShops, total := database.SearchWorkShop(searchQuery, page, database.WorkshopLimit, sort)

		users := make([]string, 0)
		for _, workshop := range workShops {
			users = appendIfMissing(users, workshop.Author)
		}

		mapUser := make(map[string]database.User)
		for _, user := range database.BulkFindUser(users...) {
			mapUser[user.BattleID] = user
		}

		q := helpers.ToURLQuery(urlQuery).Encode()
		pager := paging.New(total, database.WorkshopLimit, page, "/?"+q)
		ctx.ViewData("Pager", pager)

		ctx.ViewData("Workshops", workShops)
		ctx.ViewData("Authors", mapUser)
		ctx.ViewData("Total", len(workShops))
		ctx.ViewData("sort", sort)
		ctx.View("index.tmpl")
	})

	routes.NewAuthRoutes(app.Party("/auth"))
	routes.NewUserRoutes(app.Party("/u"))
	routes.NewViewingRoutes(app.Party("/"))

	app.Run(iris.Addr(*listen))
}

func appendIfMissing(slice []string, i string) []string {
	for _, ele := range slice {
		if ele == i {
			return slice
		}
	}
	return append(slice, i)
}
