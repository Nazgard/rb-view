package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"text/template"
	"time"
)

type Departure struct {
	Time time.Time
	Name string
}

var httpClient = &http.Client{
	Timeout: time.Second * 10,
}

var state = make(map[string]string)

func loadTimes() map[string]string {
	resp, err := httpClient.Get("http://192.144.59.250:5000/api/deaths")
	if err != nil {
		log.Println("Error getting deaths: ", err)
		return make(map[string]string)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result map[string]string
	_ = json.Unmarshal(body, &result)
	return result
}

func updateState() {
	state = loadTimes()
}

func main() {
	go func() {
		for {
			updateState()
			time.Sleep(1 * time.Second)
		}
	}()

	http.HandleFunc("/", handler)
	fmt.Println("Server starting on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func handler(w http.ResponseWriter, r *http.Request) {
	departures := make([]Departure, 0)
	for name, t := range state {
		parsedTime, err := time.Parse("2006-01-02 15:04:05", t)
		if err != nil {
			continue
		}
		departures = append(departures, Departure{parsedTime, name})
	}

	// Сортируем по времени по возрастанию
	sort.Slice(departures, func(i, j int) bool {
		return departures[i].Time.Before(departures[j].Time)
	})

	tmpl := `
<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta http-equiv="refresh" content="5">
    <title>Airport Departures</title>
    <style>
        body {
            background-color: #333;
            color: #fff;
            font-family: monospace;
            text-align: center;
        }
        h1 {
            color: #ffcc00;
        }
        table {
            margin: 0 auto;
            border-collapse: collapse;
            width: 60%;
            background-color: #000;
        }
        th, td {
            padding: 10px;
            text-align: left;
            border: 1px solid #555;
            color: #fff;
            font-size: 1.2em;
        }
        th {
            background-color: #444;
        }
        th:first-child {
            color: #ffcc00;
        }
    </style>
</head>
<body>
    <h1>Полевые боссы сервера Айрин</h1>
    <table>
        <tr>
            <th>Время</th>
            <th>Босс</th>
        </tr>
        {{range .}}
        <tr>
            <td>{{.Time.Format "02.01 15:04:05"}} МСК</td>
            <td>{{.Name}}</td>
        </tr>
        {{end}}
    </table>
</body>
</html>
	`

	t, err := template.New("departures").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = t.Execute(w, departures)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
