package main

import (
	"database/sql"
	"fmt"
	"syscall/js"

	_ "github.com/ncruces/go-sqlite3/driver"

	"github.com/example/go-wasm-sqlite-poc/internal/database"
)

var db *sql.DB
var defaultResolver string

func main() {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err == nil {
		err = database.InitSchema(db)
	}
	if err == nil {
		err = database.SeedDB(db)
	}
	if err == nil {
		defaultResolver, err = database.GetDefaultResolver()
	}
	if err != nil {
		fmt.Println("db init error:", err)
		if db != nil {
			db.Close()
			db = nil
		}
	}

	js.Global().Set("goReady", js.FuncOf(handleReady))
	js.Global().Set("goCreateShortcut", js.FuncOf(handleCreateShortcut))
	js.Global().Set("goListShortcuts", js.FuncOf(handleListShortcuts))
	js.Global().Set("goUpdateShortcut", js.FuncOf(handleUpdateShortcut))
	js.Global().Set("goDeleteShortcut", js.FuncOf(handleDeleteShortcut))
	js.Global().Set("goResolveShortcut", js.FuncOf(handleResolveShortcut))
	js.Global().Set("goSearchShortcuts", js.FuncOf(handleSearchShortcuts))

	func() {
		defer func() { recover() }()
		if fn := js.Global().Get("onGoReady"); fn.Truthy() && fn.Type() == js.TypeFunction {
			fn.Invoke()
		}
	}()

	select {}
}

func errMap(msg string) map[string]interface{} {
	return map[string]interface{}{"error": msg}
}

func handleReady(this js.Value, args []js.Value) interface{} {
	var cb js.Value
	if len(args) > 0 {
		cb = args[0]
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		return true
	})
	return nil
}

func handleCreateShortcut(this js.Value, args []js.Value) interface{} {
	name := ""
	urlStr := ""
	searchURL := ""
	desc := ""
	var cb js.Value
	if len(args) > 0 {
		name = args[0].String()
	}
	if len(args) > 1 {
		urlStr = args[1].String()
	}
	if len(args) > 2 {
		if args[2].Type() == js.TypeFunction {
			cb = args[2]
		} else if args[2].Type() == js.TypeString {
			searchURL = args[2].String()
			if len(args) > 3 {
				if args[3].Type() == js.TypeFunction {
					cb = args[3]
				} else if args[3].Type() == js.TypeString {
					desc = args[3].String()
					if len(args) > 4 {
						cb = args[4]
					}
				}
			}
		}
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		s, err := database.CreateShortcut(db, name, urlStr, searchURL, desc)
		if err != nil {
			return errMap(err.Error())
		}
		return map[string]interface{}{
			"id":          s.ID,
			"name":        s.Name,
			"url":         s.URL,
			"search_url":  s.SearchURL,
			"description": s.Description,
			"created_at":  s.CreatedAt,
		}
	})
	return nil
}

func handleListShortcuts(this js.Value, args []js.Value) interface{} {
	var cb js.Value
	if len(args) > 0 {
		cb = args[0]
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		shortcuts, err := database.ListShortcuts(db)
		if err != nil {
			return errMap(err.Error())
		}
		out := []interface{}{}
		for _, s := range shortcuts {
			out = append(out, map[string]interface{}{
				"id":          s.ID,
				"name":        s.Name,
				"url":         s.URL,
				"search_url":  s.SearchURL,
				"description": s.Description,
				"created_at":  s.CreatedAt,
			})
		}
		return out
	})
	return nil
}

func handleUpdateShortcut(this js.Value, args []js.Value) interface{} {
	id := 0
	name := ""
	urlStr := ""
	searchURL := ""
	desc := ""
	var cb js.Value
	if len(args) > 0 {
		id = args[0].Int()
	}
	if len(args) > 1 {
		name = args[1].String()
	}
	if len(args) > 2 {
		urlStr = args[2].String()
	}
	if len(args) > 3 {
		if args[3].Type() == js.TypeFunction {
			cb = args[3]
		} else if args[3].Type() == js.TypeString {
			searchURL = args[3].String()
			if len(args) > 4 {
				if args[4].Type() == js.TypeFunction {
					cb = args[4]
				} else if args[4].Type() == js.TypeString {
					desc = args[4].String()
					if len(args) > 5 {
						cb = args[5]
					}
				}
			}
		}
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		s, err := database.UpdateShortcut(db, id, name, urlStr, searchURL, desc)
		if err != nil {
			return errMap(err.Error())
		}
		var updatedAt string
		if err := db.QueryRow("SELECT datetime('now')").Scan(&updatedAt); err != nil {
			return errMap(err.Error())
		}
		return map[string]interface{}{
			"id":          s.ID,
			"name":        s.Name,
			"url":         s.URL,
			"search_url":  s.SearchURL,
			"description": s.Description,
			"updated_at":  updatedAt,
		}
	})
	return nil
}

func handleDeleteShortcut(this js.Value, args []js.Value) interface{} {
	id := 0
	var cb js.Value
	if len(args) > 0 {
		id = args[0].Int()
	}
	if len(args) > 1 {
		cb = args[1]
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		if err := database.DeleteShortcut(db, id); err != nil {
			return errMap(err.Error())
		}
		return map[string]interface{}{"id": id}
	})
	return nil
}

func handleResolveShortcut(this js.Value, args []js.Value) interface{} {
	name := ""
	var cb js.Value
	if len(args) > 0 {
		name = args[0].String()
	}
	if len(args) > 1 {
		cb = args[1]
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		u, searchURL, err := database.ResolveShortcut(db, name)
		if err != nil {
			// If shortcut not found and default resolver is configured, try resolving with default resolver
			if err == database.ErrNotFound && defaultResolver != "" {
				defaultName := defaultResolver + " " + name
				u, searchURL, err = database.ResolveShortcut(db, defaultName)
				if err == nil {
					// Successfully resolved with default resolver
					return map[string]interface{}{"url": u, "search_url": searchURL}
				}
				// If default resolver also fails, return the original not found error
				err = database.ErrNotFound
			}
			return errMap(err.Error())
		}
		return map[string]interface{}{"url": u, "search_url": searchURL}
	})
	return nil
}

func handleSearchShortcuts(this js.Value, args []js.Value) interface{} {
	query := ""
	var cb js.Value
	if len(args) > 0 {
		query = args[0].String()
	}
	if len(args) > 1 {
		cb = args[1]
	}
	runSafe(cb, func() interface{} {
		if db == nil {
			return errMap("database not initialized")
		}
		shortcuts, err := database.SearchShortcuts(db, query)
		if err != nil {
			return errMap(err.Error())
		}
		out := []interface{}{}
		for _, s := range shortcuts {
			out = append(out, map[string]interface{}{
				"id":          s.ID,
				"name":        s.Name,
				"url":         s.URL,
				"search_url":  s.SearchURL,
				"description": s.Description,
				"created_at":  s.CreatedAt,
			})
		}
		return out
	})
	return nil
}

func runSafe(cb js.Value, fn func() interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if cb.Truthy() {
				cb.Invoke(map[string]interface{}{"error": fmt.Sprintf("panic: %v", r)})
			}
		}
	}()
	result := fn()
	if cb.Truthy() {
		cb.Invoke(result)
	}
}
