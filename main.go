package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
)

type Input struct {
	Pc          float64
	B           float64
	Sigma1      float64
	Sigma2      float64
	Tolerance   float64
	HoursPerDay float64
}

type ScenarioResult struct {
	Sigma      float64
	DeltaW     float64
	WGood      float64
	WBad       float64
	Profit     float64
	Penalty    float64
	Net        float64
	LowerBound float64
	UpperBound float64
}

type Result struct {
	In          Input
	Scenario1   ScenarioResult
	Scenario2   ScenarioResult
	Improvement float64
}

type PageData struct {
	In        Input
	Prefill   bool
	HasResult bool
	Res       Result
	Error     string
}

func parseFloat(r *http.Request, name string) (float64, error) {
	return strconv.ParseFloat(r.FormValue(name), 64)
}

func normalCDF(x float64) float64 {
	return 0.5 * (1.0 + math.Erf(x/math.Sqrt2))
}

func calcDeltaW(pc, sigma, lower, upper float64) float64 {
	if sigma <= 0 {
		if pc >= lower && pc <= upper {
			return 1
		}
		return 0
	}
	z1 := (lower - pc) / sigma
	z2 := (upper - pc) / sigma
	return normalCDF(z2) - normalCDF(z1)
}

func calcScenario(in Input, sigma float64) ScenarioResult {
	toleranceMW := in.Pc * in.Tolerance / 100.0
	lower := in.Pc - toleranceMW
	upper := in.Pc + toleranceMW

	deltaW := calcDeltaW(in.Pc, sigma, lower, upper)

	totalEnergy := in.Pc * in.HoursPerDay
	wGood := totalEnergy * deltaW
	wBad := totalEnergy * (1.0 - deltaW)

	profit := wGood * in.B
	penalty := wBad * in.B
	net := profit - penalty

	return ScenarioResult{
		Sigma:      sigma,
		DeltaW:     deltaW,
		WGood:      wGood,
		WBad:       wBad,
		Profit:     profit,
		Penalty:    penalty,
		Net:        net,
		LowerBound: lower,
		UpperBound: upper,
	}
}

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"mul100": func(x float64) float64 { return x * 100.0 },
}).Parse(`
<!doctype html>
<html lang="uk">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Практична №3 — Калькулятор прибутку СЕС</title>
  <style>
    :root {
      --bg:#0b0f19; --card:#121a2b; --text:#e7eefc; --muted:#9db0d0;
      --line:#23304a; --accent:#7aa2ff; --bad:#ff6b6b; --good:#2ee59d;
    }
    * { box-sizing: border-box; }
    body {
      margin:0;
      font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Arial;
      background: var(--bg);
      color: var(--text);
    }
    .wrap { max-width: 1100px; margin: 0 auto; padding: 28px 16px 60px; }
    .title h1 { margin:0; font-size:26px; }
    .title p { margin:6px 0 0; color:var(--muted); line-height:1.45; }
    .card {
      margin-top:18px;
      background:rgba(18,26,43,0.88);
      border:1px solid rgba(35,48,74,0.75);
      box-shadow:0 10px 30px rgba(0,0,0,0.35);
      border-radius:18px;
      padding:18px;
    }
    .grid { display:grid; gap:12px; grid-template-columns:repeat(1,minmax(0,1fr)); }
    @media (min-width: 720px) { .grid { grid-template-columns:repeat(2,minmax(0,1fr)); } }
    .field { display:flex; flex-direction:column; gap:6px; }
    label { color:var(--muted); font-size:13px; }
    input[type=number] {
      width:100%;
      padding:10px 12px;
      border-radius:12px;
      border:1px solid rgba(35,48,74,0.9);
      background:rgba(7,10,18,0.55);
      color:var(--text);
      outline:none;
    }
    .actions { display:flex; gap:10px; flex-wrap:wrap; margin-top:14px; }
    button {
      border:0;
      padding:11px 14px;
      border-radius:12px;
      cursor:pointer;
      font-weight:600;
    }
    .primary { background:var(--accent); color:#0b0f19; }
    .ghost { background:rgba(255,255,255,0.06); color:var(--text); border:1px solid rgba(35,48,74,0.9); }
    table {
      width:100%;
      border-collapse:collapse;
      overflow:hidden;
      border-radius:14px;
      border:1px solid rgba(35,48,74,0.9);
      margin-top:10px;
    }
    th, td { padding:10px; border-bottom:1px solid rgba(35,48,74,0.75); text-align:right; }
    th { text-align:left; color:var(--muted); background:rgba(255,255,255,0.04); }
    tr:last-child td { border-bottom:0; }
    .rowhead { text-align:left; }
    .two { display:grid; gap:12px; grid-template-columns:1fr; }
    @media (min-width: 900px) { .two { grid-template-columns:1fr 1fr; } }
    .pill {
      display:inline-flex;
      gap:8px;
      align-items:center;
      padding:6px 10px;
      border-radius:999px;
      border:1px solid rgba(35,48,74,0.9);
      background:rgba(255,255,255,0.05);
      color:var(--muted);
      font-size:12px;
      margin-right:8px;
      margin-bottom:8px;
    }
    .error { border:1px solid rgba(255,107,107,0.55); background:rgba(255,107,107,0.08); }
    .good { color: var(--good); font-weight: 700; }
    .bad { color: var(--bad); font-weight: 700; }
    .note { color:var(--muted); font-size:13px; line-height:1.45; }
  </style>
</head>
<body>
<div class="wrap">
  <div class="title">
    <h1>Кушнір Катерина Вікторівна — ТВз-21</h1>
    <p>Практична робота №3 • Веб-калькулятор розрахунку прибутку від сонячної електростанції з встановленою системою прогнозування сонячної потужності</p>
  </div>

  {{if .Error}}
    <div class="card error"><b>Помилка:</b> {{.Error}}</div>
  {{end}}

  {{if .HasResult}}
    <div class="card">
      <h2>Результати</h2>

      <div style="margin-top:10px;">
        <span class="pill"><b>Pc</b> = {{printf "%.2f" .Res.In.Pc}} МВт</span>
        <span class="pill"><b>B</b> = {{printf "%.2f" .Res.In.B}} грн/кВт·год</span>
        <span class="pill"><b>Δдоп</b> = ±{{printf "%.2f" .Res.In.Tolerance}}%</span>
        <span class="pill"><b>Покращення балансу</b> = {{printf "%.2f" .Res.Improvement}} тис. грн</span>
      </div>

      <div class="two">
        <div>
          <h3>Сценарій 1 — початкова похибка</h3>
          <table>
            <tr><th>Показник</th><th>Значення</th></tr>
            <tr><td class="rowhead">σ₁, МВт</td><td>{{printf "%.2f" .Res.Scenario1.Sigma}}</td></tr>
            <tr><td class="rowhead">Допустимий діапазон, МВт</td><td>{{printf "%.2f" .Res.Scenario1.LowerBound}} – {{printf "%.2f" .Res.Scenario1.UpperBound}}</td></tr>
            <tr><td class="rowhead">δw₁, %</td><td>{{printf "%.2f" (mul100 .Res.Scenario1.DeltaW)}}</td></tr>
            <tr><td class="rowhead">W₁, МВт·год</td><td>{{printf "%.2f" .Res.Scenario1.WGood}}</td></tr>
            <tr><td class="rowhead">W₂, МВт·год</td><td>{{printf "%.2f" .Res.Scenario1.WBad}}</td></tr>
            <tr><td class="rowhead">П₁, тис. грн</td><td>{{printf "%.2f" .Res.Scenario1.Profit}}</td></tr>
            <tr><td class="rowhead">Ш₁, тис. грн</td><td>{{printf "%.2f" .Res.Scenario1.Penalty}}</td></tr>
            <tr><td class="rowhead"><b>Баланс, тис. грн</b></td>
              <td class="{{if ge .Res.Scenario1.Net 0.0}}good{{else}}bad{{end}}">{{printf "%.2f" .Res.Scenario1.Net}}</td></tr>
          </table>
        </div>

        <div>
          <h3>Сценарій 2 — після покращення прогнозу</h3>
          <table>
            <tr><th>Показник</th><th>Значення</th></tr>
            <tr><td class="rowhead">σ₂, МВт</td><td>{{printf "%.2f" .Res.Scenario2.Sigma}}</td></tr>
            <tr><td class="rowhead">Допустимий діапазон, МВт</td><td>{{printf "%.2f" .Res.Scenario2.LowerBound}} – {{printf "%.2f" .Res.Scenario2.UpperBound}}</td></tr>
            <tr><td class="rowhead">δw₂, %</td><td>{{printf "%.2f" (mul100 .Res.Scenario2.DeltaW)}}</td></tr>
            <tr><td class="rowhead">W₃, МВт·год</td><td>{{printf "%.2f" .Res.Scenario2.WGood}}</td></tr>
            <tr><td class="rowhead">W₄, МВт·год</td><td>{{printf "%.2f" .Res.Scenario2.WBad}}</td></tr>
            <tr><td class="rowhead">П₂, тис. грн</td><td>{{printf "%.2f" .Res.Scenario2.Profit}}</td></tr>
            <tr><td class="rowhead">Ш₂, тис. грн</td><td>{{printf "%.2f" .Res.Scenario2.Penalty}}</td></tr>
            <tr><td class="rowhead"><b>Баланс, тис. грн</b></td>
              <td class="{{if ge .Res.Scenario2.Net 0.0}}good{{else}}bad{{end}}">{{printf "%.2f" .Res.Scenario2.Net}}</td></tr>
          </table>
        </div>
      </div>

      <div class="two">
        <div>
          <h3>Контрольний приклад</h3>
          <table>
            <tr><th>Показник</th><th>Очікувано</th><th>Отримано</th></tr>
            <tr><td class="rowhead">δw₁, %</td><td>20.00</td><td>{{printf "%.2f" (mul100 .Res.Scenario1.DeltaW)}}</td></tr>
            <tr><td class="rowhead">W₁, МВт·год</td><td>24.00</td><td>{{printf "%.2f" .Res.Scenario1.WGood}}</td></tr>
            <tr><td class="rowhead">П₁, тис. грн</td><td>168.00</td><td>{{printf "%.2f" .Res.Scenario1.Profit}}</td></tr>
            <tr><td class="rowhead">Ш₁, тис. грн</td><td>672.00</td><td>{{printf "%.2f" .Res.Scenario1.Penalty}}</td></tr>
            <tr><td class="rowhead">Баланс 1, тис. грн</td><td>-504.00</td><td>{{printf "%.2f" .Res.Scenario1.Net}}</td></tr>
            <tr><td class="rowhead">δw₂, %</td><td>68.00</td><td>{{printf "%.2f" (mul100 .Res.Scenario2.DeltaW)}}</td></tr>
            <tr><td class="rowhead">W₃, МВт·год</td><td>81.60</td><td>{{printf "%.2f" .Res.Scenario2.WGood}}</td></tr>
          </table>
        </div>

        <div>
          <h3>Пояснення</h3>
          <p class="note">
            У практичній роботі потрібно створити веб-калькулятор для розрахунку прибутку СЕС
            за контрольним прикладом. У файлі завдання наведено саме такий приклад для
            <b>Pc = 5 МВт</b>, <b>B = 7 грн/кВт·год</b>, <b>σ₁ = 1 МВт</b> і <b>σ₂ = 0.25 МВт</b>. :contentReference[oaicite:1]{index=1}
          </p>
        </div>
      </div>
    </div>
  {{end}}

  <div class="card">
    <h2>Ввід даних</h2>
    <form method="post" action="/calculate">
      <div class="grid">
        <div class="field">
          <label>Потужність СЕС, МВт</label>
          <input name="Pc" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.Pc}}{{end}}">
        </div>
        <div class="field">
          <label>Вартість електроенергії, грн/кВт·год</label>
          <input name="B" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.B}}{{end}}">
        </div>
        <div class="field">
          <label>Початкова похибка σ₁, МВт</label>
          <input name="Sigma1" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.Sigma1}}{{end}}">
        </div>
        <div class="field">
          <label>Покращена похибка σ₂, МВт</label>
          <input name="Sigma2" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.Sigma2}}{{end}}">
        </div>
        <div class="field">
          <label>Допустиме відхилення, %</label>
          <input name="Tolerance" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.Tolerance}}{{end}}">
        </div>
        <div class="field">
          <label>Тривалість періоду, год</label>
          <input name="HoursPerDay" step="any" type="number" placeholder="Введіть значення" value="{{if .Prefill}}{{printf "%.2f" .In.HoursPerDay}}{{end}}">
        </div>
      </div>

      <div class="actions">
        <button class="primary" type="submit">Розрахувати</button>
        <button class="ghost" type="button" onclick="fillControl()">Контрольний приклад</button>
      </div>
    </form>
  </div>
</div>

<script>
function fillControl() {
  document.querySelector('[name="Pc"]').value = 5;
  document.querySelector('[name="B"]').value = 7;
  document.querySelector('[name="Sigma1"]').value = 1;
  document.querySelector('[name="Sigma2"]').value = 0.25;
  document.querySelector('[name="Tolerance"]').value = 5;
  document.querySelector('[name="HoursPerDay"]').value = 24;
}
</script>
</body>
</html>
`))

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		In:      Input{},
		Prefill: false,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Execute(w, data)
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		In:      Input{},
		Prefill: true,
	}

	if r.Method != http.MethodPost {
		data.Error = "Невірний метод запиту."
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, data)
		return
	}

	if err := r.ParseForm(); err != nil {
		data.Error = "Не вдалося прочитати форму."
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, data)
		return
	}

	var err error
	if data.In.Pc, err = parseFloat(r, "Pc"); err != nil {
		data.Error = "Pc має бути числом."
	}
	if data.In.B, err = parseFloat(r, "B"); err != nil {
		data.Error = "B має бути числом."
	}
	if data.In.Sigma1, err = parseFloat(r, "Sigma1"); err != nil {
		data.Error = "Sigma1 має бути числом."
	}
	if data.In.Sigma2, err = parseFloat(r, "Sigma2"); err != nil {
		data.Error = "Sigma2 має бути числом."
	}
	if data.In.Tolerance, err = parseFloat(r, "Tolerance"); err != nil {
		data.Error = "Tolerance має бути числом."
	}
	if data.In.HoursPerDay, err = parseFloat(r, "HoursPerDay"); err != nil {
		data.Error = "HoursPerDay має бути числом."
	}

	if data.Error != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, data)
		return
	}

	if data.In.Pc <= 0 || data.In.B <= 0 || data.In.Sigma1 <= 0 || data.In.Sigma2 <= 0 || data.In.Tolerance <= 0 || data.In.HoursPerDay <= 0 {
		data.Error = "Усі значення мають бути більшими за нуль."
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = page.Execute(w, data)
		return
	}

	res := Result{In: data.In}
	res.Scenario1 = calcScenario(data.In, data.In.Sigma1)
	res.Scenario2 = calcScenario(data.In, data.In.Sigma2)
	res.Improvement = res.Scenario2.Net - res.Scenario1.Net

	data.HasResult = true
	data.Res = res

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = page.Execute(w, data)
}

func main() {
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/calculate", handleCalculate)

	log.Println("Server started: http://localhost:9092")
	log.Fatal(http.ListenAndServe(":9092", nil))
}
