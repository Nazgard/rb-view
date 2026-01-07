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
	Time        time.Time
	Name        string
	MinutesLeft int  // минуты до/после респа (отрицательное — уже прошло)
	IsPast      bool // true если респ уже был
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
	// Определяем режим: resp — время респа (+5 часов), иначе — время смерти
	mode := r.URL.Query().Get("mode")
	showResp := mode == "resp"
	now := time.Now()

	departures := make([]Departure, 0, len(state))
	for name, t := range state {
		parsedTime, err := time.Parse("2006-01-02 15:04:05", t)
		if err != nil {
			log.Printf("Failed to parse time '%s' for %s: %v\n", t, name, err)
			continue
		}

		// Если включён режим респа — добавляем 5 часов
		if showResp {
			parsedTime = parsedTime.Add(5 * time.Hour)
		}

		minutesLeft := int(parsedTime.Sub(now).Minutes())
		isPast := minutesLeft < 0

		departures = append(departures, Departure{
			Time:        parsedTime,
			Name:        name,
			MinutesLeft: minutesLeft,
			IsPast:      isPast,
		})
	}

	// Сортировка по отображаемому времени
	sort.Slice(departures, func(i, j int) bool {
		return departures[i].Time.Before(departures[j].Time)
	})

	tmpl := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Полевые боссы сервера Айрин</title>
    <meta http-equiv="refresh" content="5">
    <style>
        body {
            background-color: #333;
            color: #fff;
            font-family: monospace;
            text-align: center;
            margin: 0;
            padding: 20px;
        }
        h1 {
            color: #ffcc00;
            margin-bottom: 10px;
        }
        .toggle-container {
            margin-bottom: 20px;
            font-size: 1.2em;
        }
        .toggle-switch {
            position: relative;
            display: inline-block;
            width: 60px;
            height: 34px;
            margin: 0 10px;
        }
        .toggle-switch input {
            opacity: 0;
            width: 0;
            height: 0;
        }
        .slider {
            position: absolute;
            cursor: pointer;
            top: 0; left: 0; right: 0; bottom: 0;
            background-color: #ccc;
            transition: .4s;
            border-radius: 34px;
        }
        .slider:before {
            position: absolute;
            content: "";
            height: 26px;
            width: 26px;
            left: 4px;
            bottom: 4px;
            background-color: white;
            transition: .4s;
            border-radius: 50%;
        }
        input:checked + .slider {
            background-color: #ffcc00;
        }
        input:checked + .slider:before {
            transform: translateX(26px);
        }
        table {
            margin: 0 auto;
            border-collapse: collapse;
            width: 70%;
            background-color: #000;
        }
        th, td {
            padding: 12px;
            text-align: left;
            border: 1px solid #555;
            color: #fff;
            font-size: 1.3em;
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

    <div class="toggle-container">
        <label>
            <strong>Показывать:</strong> время смерти
        </label>
        <label class="toggle-switch">
            <input type="checkbox" id="modeToggle" {{if .ShowResp}}checked{{end}} onchange="toggleMode()">
            <span class="slider"></span>
        </label>
        <label>
            время респа (+5 часов)
        </label>
    </div>

    <table>
        <tr>
            <th>Время</th>
            <th>Босс</th>
        </tr>
        {{range .Departures}}
        <tr>
            <td>{{.Time.Format "02.01 15:04:05"}} МСК 
				{{if not .IsPast}}
					<em>(через {{.MinutesLeft}} мин)</em>
                {{end}}
			</td>
            <td>{{.Name}}</td>
        </tr>
        {{end}}
    </table>

    <script>
        function toggleMode() {
            const checkbox = document.getElementById('modeToggle');
            const newUrl = checkbox.checked 
                ? window.location.pathname + '?mode=resp'
                : window.location.pathname;
            window.location.href = newUrl;
        }
    </script>
</body>
</html>
	`

	// Передаём данные в шаблон
	data := struct {
		Departures []Departure
		ShowResp   bool
	}{
		Departures: departures,
		ShowResp:   showResp,
	}

	t, err := template.New("departures").Parse(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
