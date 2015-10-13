package main

import (
  "bytes"
  "encoding/json"
  "log"
  "net/http"
  "strings"
  "time"
)

func main(){
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello!"))
  })

  mw := multiWeatherProvider{
    openWeatherMap{apiKey: "632b34ca0a4a4b6a47104f86a25715af"},
    weatherUnderground{apiKey: "9cb0ff664cabfbda"},
  }

  http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
    begin := time.Now()
    city := strings.SplitN(r.URL.Path, "/", 3)[2]

    temp, err := mw.temperature(city)
    if err != nil {
      http.Error(w, err.Error(), http.StatusInternalServerError)
      return
    }

    w.Header().Set("Content-Type", "application/json; charset=utf-8")
    json.NewEncoder(w).Encode(map[string]interface{}{
      "city": city,
      "temp": temp,
      "took": time.Since(begin).String(),
    })
  })

  http.ListenAndServe(":8080", nil)
}

type weatherProvider interface {
  temperature(city string) (float64, error)
}

type openWeatherMap struct {
  apiKey string
}

func (w openWeatherMap) temperature(city string) (float64, error) {
  resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?q=" + city + "&appid=" + w.apiKey)
  if err != nil {
    return 0, err
  }

  defer resp.Body.Close()

  var d struct {
    Main struct {
      Kelvin float64 `json:"temp"`
    } `json:"main"`
  }

  if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
    return 0, err
  }

  log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
  return d.Main.Kelvin, nil
}

type weatherUnderground struct {
  apiKey string
}

func (w weatherUnderground) temperature(city string) (float64, error) {
  resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")
  if err != nil {
    return 0, err
  }

  defer resp.Body.Close()

  var d struct {
    Observation struct {
      Celsius float64 `json:"temp_c"`
    } `json:"current_observation"`
  }

  if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
    buf := new(bytes.Buffer)
    buf.ReadFrom(resp.Body)
    s := buf.String()
    log.Printf(s)
    return 0, err
  }

  kelvin := d.Observation.Celsius + 273.15
  log.Printf("weatherUnderground: %s: %.2f", city, kelvin)
  return kelvin, nil
}

func temperature(city string, providers ...weatherProvider) (float64, error) {
  sum := 0.0

  for _, provider := range providers {
    k, err := provider.temperature(city)
    if err != nil {
      return 0, err
    }

    sum += k
  }

  return sum / float64(len(providers)), nil
}

type multiWeatherProvider []weatherProvider

func (w multiWeatherProvider) temperature(city string) (float64, error) {
  sum := 0.0

  for _, provider := range w {
    k, err := provider.temperature(city)
    if err != nil {
      return 0, err
    }

    sum += k
  }

  return sum / float64(len(w)), nil
}