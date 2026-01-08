package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"sort"
	"time"
)

type TableEntry struct {
	Name        string `json:"name"`
	Time        string `json:"time"`         // Формат: "02.01 15:04:05"
	MinutesLeft int    `json:"minutes_left"` // может быть отрицательным
	IsPast      bool   `json:"is_past"`
}

var (
	httpClient     = &http.Client{Timeout: 10 * time.Second}
	state          = make(map[string]string) // сырые данные смерти от внешнего API
	moscowLocation *time.Location
	tmpl           = template.Must(template.ParseFiles("templates/page.html"))
)

func init() {
	var err error
	moscowLocation, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		log.Fatal("Cannot load timezone Europe/Moscow:", err)
	}
}

func loadTimes() map[string]string {
	resp, err := httpClient.Get("http://192.144.59.250:5000/api/deaths")
	if err != nil {
		log.Println("Error getting deaths:", err)
		return make(map[string]string)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		return make(map[string]string)
	}

	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		log.Println("Error unmarshaling JSON:", err)
		return make(map[string]string)
	}
	return result
}

func updateState() {
	state = loadTimes()
}

// Новый JSON API для таблицы
func tableAPIHandler(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	showResp := mode == "resp"

	now := time.Now().In(moscowLocation)

	entries := make([]TableEntry, 0, len(state))

	for name, t := range state {
		parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", t, moscowLocation)
		if err != nil {
			log.Printf("Failed to parse time '%s' for %s: %v\n", t, name, err)
			continue
		}

		if showResp {
			parsedTime = parsedTime.Add(5 * time.Hour)
		}

		minutesLeft := int(parsedTime.Sub(now).Minutes())
		isPast := minutesLeft < 0

		entries = append(entries, TableEntry{
			Name:        name,
			Time:        parsedTime.Format("02.01 15:04:05"),
			MinutesLeft: minutesLeft,
			IsPast:      isPast,
		})
	}

	// Сортировка по времени
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Time < entries[j].Time // строка, но формат позволяет лексикографическую сортировку
	})

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(entries)
}

func main() {
	go func() {
		for {
			updateState()
			time.Sleep(1 * time.Second)
		}
	}()

	http.HandleFunc("/", handler)
	http.HandleFunc("/api/table", tableAPIHandler)
	// Статические файлы (звук, если добавишь картинки и т.д.)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static/"))))

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Начальная загрузка — просто отдаём HTML с пустой таблицей или с данными (можно пустую)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, struct{ ShowResp bool }{ShowResp: r.URL.Query().Get("mode") == "resp"}); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Println("Template execute error:", err)
	}
}
