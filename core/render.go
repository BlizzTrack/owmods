package core

import (
	"html/template"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/kataras/iris"
	"github.com/kataras/iris/view"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/css"
	"github.com/tdewolff/minify/html"
	"github.com/tdewolff/minify/js"
)

type (
	render struct {
		ViewEngine *view.HTMLEngine
		path       string
		minify     *minify.M
	}

	HeaderMetaData struct {
		Title    string
		Logo     string
		Desc     string
		SiteName string
	}
)

var Render *render

func NewRender(folder, vExt string) *render {
	localRender := iris.HTML(folder, vExt).Layout("layout.tmpl").Reload(true)
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("text/html", html.Minify)
	m.AddFunc("text/javascript", js.Minify)

	Render = &render{ViewEngine: localRender, path: folder, minify: m}

	return Render;
}

func (render *render) AddFunc(name string, method interface{}) *render {
	render.ViewEngine.AddFunc(name, method)

	return render
}

func (render *render) AddLayoutFunc(name string, method interface{}) *render {
	render.ViewEngine.AddLayoutFunc(name, method)

	return render
}

func (render *render) Defaults() *render {
	render.AddFunc("log", func(format string, params ...string) string {
		newKeys := make([]interface{}, len(params))
		for i, v := range params {
			newKeys[i] = v
		}

		log.Printf(format, newKeys...)
		return ""
	})

	render.AddFunc("replace", func(from string, to string, word string, number int) string {
		return strings.Replace(word, from, to, number)
	})

	render.AddFunc("longerThan", func(a string, b int) bool {
		return len(a) > b
	})

	render.AddFunc("raw", func(a string) template.HTML {
		tmpl, _ := render.minify.String("text/html", a)

		return template.HTML(tmpl)
	})

	/*
		render.AddFunc("include", func(filePath string) template.HTML {
			if !Core.Settings.IsDebug() {
				cachekey := fmt.Sprintf("%s_%s", filePath, "include_no_render")
				data, err := CacheManager.Get(cachekey)
				if err != memcache.ErrCacheMiss {
					return template.HTML(data.Value)
				}

				tmpl := AssetManager.FileContents(path.Join(render.path, filePath))
				tmpl, _ = render.minify.String("text/html", tmpl)

				CacheManager.SetTimeout(cachekey, tmpl, 60)

				return template.HTML(tmpl)
			}

			tmpl := AssetManager.FileContents(path.Join(render.path, filePath))
			tmpl, _ = render.minify.String("text/html", tmpl)

			// return template.HTML(tmpl)
			return template.HTML(tmpl)
		})
	*/

	render.AddFunc("formatUTC", func(a int64) string {
		if a <= 0 {
			return "N/A"
		}

		tm := time.Unix(0, a*1000000)
		return tm.UTC().Format(time.RFC1123)
	})

	render.AddFunc("eq", func(a interface{}, b interface{}) bool {
		return a == b
	})

	render.AddFunc("neq", func(a interface{}, b interface{}) bool {
		return a != b
	})

	render.AddFunc("gr", func(a int, b int) bool {
		return a > b
	})

	render.AddFunc("lt", func(a int, b int) bool {
		return a < b
	})

	render.AddFunc("TrimSpace", func(body string) string {
		return strings.TrimSpace(body)
	})

	render.AddFunc("ToUpper", func(item string) string {
		return strings.ToUpper(item)
	})

	render.AddFunc("ToLower", func(item string) string {
		return strings.ToLower(item)
	})

	render.AddFunc("ToTitle", func(item string) string {
		return strings.ToTitle(item)
	})

	render.AddFunc("Join", func(joiner string, args ...string) string {
		return strings.Join(args, joiner)
	})

	render.AddFunc("int2str", strconv.Itoa)

	render.AddFunc("unix", func(t time.Time) int64 {
		return int64(t.UnixNano() / 1000000)
	})

	render.AddFunc("unixString", func(s string) int64 {
		layout := "2006-01-02 15:04:05 -0700 MST"
		t, err := time.Parse(layout, s)
		if err != nil {
			return -1
		}

		return t.UnixNano() / 1000000
	})

	render.AddFunc("mod", func(i, j int) int { return i%j })
	return render
}

func (render *render) Header(title string, logo string, desc string) HeaderMetaData {
	return HeaderMetaData{
		title, logo, desc, "OWMods",
	}
}
