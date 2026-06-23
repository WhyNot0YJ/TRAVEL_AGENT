package eino

import (
	"context"
	"fmt"
	"strings"
)

type RealWeatherTool struct {
	client   *amapClient
	geocoder *amapClient
}

func (t RealWeatherTool) Run(ctx context.Context, input WeatherToolInput) ([]MockWeather, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if input.City == "" {
		return nil, fmt.Errorf("weather city is required")
	}
	if input.Days <= 0 {
		return nil, fmt.Errorf("weather days must be positive")
	}

	adcode, err := t.lookupAdcode(ctx, input.City)
	if err != nil {
		return nil, err
	}
	var resp amapWeatherResponse
	if err := t.client.get(ctx, "/weather/weatherInfo", map[string]string{
		"city":       adcode,
		"extensions": "all",
	}, &resp); err != nil {
		return nil, err
	}
	if len(resp.Forecasts) == 0 || len(resp.Forecasts[0].Casts) == 0 {
		return nil, fmt.Errorf("amap weather response is empty")
	}

	casts := resp.Forecasts[0].Casts
	weather := make([]MockWeather, 0, input.Days)
	for day := 1; day <= input.Days; day++ {
		cast := casts[(day-1)%len(casts)]
		condition := cast.DayWeather
		if condition == "" {
			condition = cast.NightWeather
		}
		if condition == "" {
			condition = "unknown"
		}
		temp := cast.NightTemp + "-" + cast.DayTemp + "C"
		if cast.NightTemp == "" || cast.DayTemp == "" {
			temp = "unknown"
		}
		suggestion := "weather data from amap"
		if conditionContainsRain(condition) {
			suggestion = "keep indoor backup plan"
		}
		weather = append(weather, MockWeather{
			Day:         day,
			Condition:   condition,
			Temperature: temp,
			Suggestion:  suggestion,
		})
	}
	return weather, nil
}

func (t RealWeatherTool) lookupAdcode(ctx context.Context, city string) (string, error) {
	client := t.geocoder
	if client == nil {
		client = t.client
	}
	var resp amapGeocodeResponse
	if err := client.get(ctx, "/geocode/geo", map[string]string{
		"address": city,
	}, &resp); err != nil {
		return "", err
	}
	if len(resp.Geocodes) == 0 || resp.Geocodes[0].Adcode == "" {
		return "", fmt.Errorf("amap geocode response missing adcode")
	}
	return resp.Geocodes[0].Adcode, nil
}

type amapGeocodeResponse struct {
	Status   string            `json:"status"`
	Info     string            `json:"info"`
	Geocodes []amapGeocodeItem `json:"geocodes"`
}

type amapGeocodeItem struct {
	Adcode string `json:"adcode"`
}

type amapWeatherResponse struct {
	Status    string                `json:"status"`
	Info      string                `json:"info"`
	Forecasts []amapWeatherForecast `json:"forecasts"`
}

type amapWeatherForecast struct {
	Casts []amapWeatherCast `json:"casts"`
}

type amapWeatherCast struct {
	DayWeather   string `json:"dayweather"`
	NightWeather string `json:"nightweather"`
	DayTemp      string `json:"daytemp"`
	NightTemp    string `json:"nighttemp"`
}

func conditionContainsRain(condition string) bool {
	for _, token := range []string{"雨", "rain", "Rain", "RAIN"} {
		if strings.Contains(condition, token) {
			return true
		}
	}
	return false
}
