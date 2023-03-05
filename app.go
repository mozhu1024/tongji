package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	opt, _ := redis.ParseURL(fmt.Sprintf("redis://:%s@%s:6379", os.Getenv("PASSWORD"), os.Getenv("ADDR")))
	rdb := redis.NewClient(opt)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_id := r.URL.Query().Get("id")
		if _id == "" {
			w.Write([]byte("Err"))
			return
		}
		// 计数
		_ = rdb.Incr(ctx, _id).Err()
		// 保存信息
		infoKey := fmt.Sprintf("_%s", _id)
		if rdb.Exists(ctx, infoKey).Val() == 0 {
			_ = rdb.HSet(ctx, infoKey,
				map[string]interface{}{
					"ver":  r.URL.Query().Get("ver"),
					"os":   r.URL.Query().Get("os"),
					"arch": r.URL.Query().Get("arch"),
					"ln":   r.URL.Query().Get("ln"),
					"name": r.URL.Query().Get("name"),
				},
			)
		}
		w.Write([]byte("OK"))
	})

	http.HandleFunc("/view", func(w http.ResponseWriter, r *http.Request) {
		// scan
		var cursor uint64
		var keys = []string{}
		for {
			var _keys []string
			var err error
			_keys, cursor, err = rdb.Scan(ctx, cursor, "_*", 50).Result()
			if err != nil {
				break
			}
			keys = append(keys, _keys...)
			if cursor == 0 {
				break
			}
		}
		data := []map[string]string{}
		for _, k := range keys {
			res, err := rdb.HGetAll(ctx, k).Result()
			if err != nil {
				continue
			}
			n, err := rdb.Get(ctx, k[1:]).Result()
			if err == nil {
				res["total"] = n
			}
			data = append(data, res)
		}
		if len(r.URL.Query().Get("json")) > 0 {
			buf, err := json.Marshal(data)
			if err != nil {
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(buf)
			return
		}
		table := `<html><body>
		<h2>统计数据</h2>
		<hr/>
<table border="1" align="center" width="100%">
	<tr>
		<td>Ver</td>
		<td>OS</td>
		<td>Arch</td>
		<td>Lang</td>
		<td>Name</td>
		<td>Total</td>
	</tr>
	{{ range $v := . }}
	<tr>
		<td>{{$v.ver}}</td>
		<td>{{$v.os}}</td>
		<td>{{$v.arch}}</td>
		<td>{{$v.ln}}</td>
		<td>{{$v.name}}</td>
		<td>{{$v.total}}</td>
	</tr>
	{{end}}
</table>
</body>
</html>`
		tmpl, err := template.New("table").Parse(table)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			w.Write([]byte(err.Error()))
			return
		}
	})

	log.Printf("Listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
