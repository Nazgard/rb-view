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

type Departure struct {
	Time        time.Time
	Name        string
	MinutesLeft int
	IsPast      bool
}

var (
	httpClient     = &http.Client{Timeout: 10 * time.Second}
	state          = make(map[string]string)
	moscowLocation *time.Location
	tmpl           = template.Must(template.New("page").Parse(pageTemplate))
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

func main() {
	go func() {
		for {
			updateState()
			time.Sleep(1 * time.Second)
		}
	}()

	http.HandleFunc("/", handler)
	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	showResp := mode == "resp"

	now := time.Now().In(moscowLocation)

	departures := make([]Departure, 0, len(state))
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

		departures = append(departures, Departure{
			Time:        parsedTime,
			Name:        name,
			MinutesLeft: minutesLeft,
			IsPast:      isPast,
		})
	}

	sort.Slice(departures, func(i, j int) bool {
		return departures[i].Time.Before(departures[j].Time)
	})

	data := struct {
		Departures []Departure
		ShowResp   bool
	}{
		Departures: departures,
		ShowResp:   showResp,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		log.Println("Template execute error:", err)
	}
}

const pageTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Полевые боссы сервера Айрин</title>
    <style>
        :root {
            --bg: #1e1e1e;
            --text: #e0e0e0;
            --accent: #ffcc00;
            --table-bg: #111;
            --border: #444;
            --header-bg: #333;
        }

        html, body {
            height: 100%;
            margin: 0;
            padding: 0;
            background-color: var(--bg);
            color: var(--text);
            font-family: 'Segoe UI', 'Roboto', system-ui, sans-serif;
        }

        body {
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 2rem 2rem 2rem; /* равномерный padding, снизу чуть больше для дыхания */
            box-sizing: border-box;
        }

        h1 {
            color: var(--accent);
            font-size: 2.8rem;
            margin-bottom: 1.5rem;
            text-shadow: 0 0 10px rgba(255,204,0,0.3);
            text-align: center;
        }

        .toggle-container {
            margin-bottom: 2.5rem;
            font-size: 1.4rem;
            display: flex;
            align-items: center;
            gap: 1rem;
            flex-wrap: wrap;
            justify-content: center;
        }

        .toggle-switch {
            position: relative;
            display: inline-block;
            width: 70px;
            height: 38px;
            flex-shrink: 0;
        }

        .toggle-switch input {
            opacity: 0;
            width: 0;
            height: 0;
        }

        .slider {
            position: absolute;
            cursor: pointer;
            inset: 0;
            background-color: #555;
            transition: .4s;
            border-radius: 38px;
        }

        .slider:before {
            position: absolute;
            content: "";
            height: 30px;
            width: 30px;
            left: 4px;
            bottom: 4px;
            background-color: white;
            transition: .4s;
            border-radius: 50%;
        }

        input:checked + .slider {
            background-color: var(--accent);
        }

        input:checked + .slider:before {
            transform: translateX(32px);
        }

        .table-container {
            width: 100%;
            max-width: 1400px;
            overflow-x: auto;
            box-shadow: 0 8px 32px rgba(0,0,0,0.6);
            border-radius: 12px;
            overflow: hidden;
            background-color: var(--table-bg); /* явный bg, чтобы не было прозрачности */
        }

        table {
            width: 100%;
            min-width: 600px;
            border-collapse: collapse;
            background-color: var(--table-bg);
            font-size: 1.5rem;
        }

        th, td {
            padding: 1.2rem 1.5rem;
            text-align: left;
            border-bottom: 1px solid var(--border);
        }

        th {
            background-color: var(--header-bg);
            color: var(--accent);
            font-weight: 600;
            position: sticky;
            top: 0;
        }

        tr:hover {
            background-color: rgba(255,204,0,0.05);
        }

        .time-cell {
            white-space: nowrap;
            font-family: 'Courier New', monospace;
            font-weight: 500;
        }

        .minutes {
            color: #a0ffa0;
            font-size: 0.9em;
            margin-left: 0.8rem;
        }

        .past {
            color: #888;
        }

        .past .minutes {
            color: #666;
        }

        /* Мобильные устройства */
        @media (max-width: 768px) {
            body { padding: 1rem 1rem 1.5rem; }
            h1 { font-size: 2rem; margin-bottom: 1rem; }
            .toggle-container {
                font-size: 1.1rem;
                gap: 0.8rem;
                flex-direction: column;
                text-align: center;
            }
            .toggle-switch { width: 60px; height: 34px; }
            .slider:before { width: 26px; height: 26px; }
            input:checked + .slider:before { transform: translateX(26px); }
            table { font-size: 1.1rem; min-width: 500px; }
            th, td { padding: 0.8rem 1rem; }
            .minutes {
                display: block;
                margin-left: 0;
                margin-top: 0.3rem;
            }
        }

        @media (max-width: 480px) {
            h1 { font-size: 1.7rem; }
            .toggle-container { font-size: 1rem; }
            table { font-size: 1rem; }
            th, td { padding: 0.6rem 0.8rem; }
        }

        /* Большие экраны */
        @media (min-width: 1920px) {
            body { padding: 3rem 3rem 3rem; }
            h1 { font-size: 3.8rem; margin-bottom: 2rem; }
            .toggle-container { font-size: 1.8rem; gap: 1.5rem; }
            .toggle-switch { width: 90px; height: 48px; }
            .slider:before { width: 38px; height: 38px; }
            input:checked + .slider:before { transform: translateX(42px); }
            table { font-size: 2rem; }
            th, td { padding: 1.8rem 2rem; }
            .table-container { max-width: 1800px; }
        }

        @media (min-width: 2560px) {
            h1 { font-size: 4.5rem; }
            table { font-size: 2.4rem; }
            th, td { padding: 2.2rem 2.5rem; }
        }
    </style>
</head>
<body>
    <h1>Полевые боссы сервера Айрин</h1>

    <div class="toggle-container">
        <strong>Показывать:</strong>
        <span>время смерти</span>
        <label class="toggle-switch">
            <input type="checkbox" id="modeToggle" {{if .ShowResp}}checked{{end}} onchange="toggleMode()">
            <span class="slider"></span>
        </label>
        <span>время респа (+5 ч)</span>
    </div>

    <div class="table-container">
        <table>
            <thead>
                <tr>
                    <th>Время (МСК)</th>
                    <th>Босс</th>
                </tr>
            </thead>
            <tbody>
                {{range .Departures}}
                <tr {{if .IsPast}}class="past"{{end}}>
                    <td class="time-cell">
                        {{.Time.Format "02.01 15:04:05"}}
                        {{if not .IsPast}}
                            <span class="minutes">(через {{.MinutesLeft}} мин)</span>
                        {{end}}
                    </td>
                    <td>{{.Name}}</td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>

    <script>
        function toggleMode() {
            const checked = document.getElementById('modeToggle').checked;
            const url = checked ? '?mode=resp' : '/';
            if (window.location.search !== (checked ? '?mode=resp' : '')) {
                window.location.href = url;
            }
        }

        // Автообновление каждые 10 секунд
        setInterval(() => {
            fetch(window.location.href)
                .then(r => r.text())
                .then(html => {
                    document.open();
                    document.write(html);
                    document.close();
                })
                .catch(err => console.error('Update failed:', err));
        }, 10000);
    </script>
</body>
</html>`
