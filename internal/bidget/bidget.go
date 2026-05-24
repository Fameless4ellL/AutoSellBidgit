package bidget

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	GET       string = "GET"
	POST      string = "POST"
	EmptyBody string = "{}"
)

const (
	Assets         string = "/api/v2/spot/account/assets"
	PlaceOrder     string = "/api/v2/spot/trade/place-order"
	SymbolInfo     string = "/api/v2/spot/public/symbols"
	UnfilledOrders string = "/api/v2/spot/trade/unfilled-orders"
)

var client *http.Client = &http.Client{Timeout: 5 * time.Second}

func generateSignature(timestamp, method, requestPath, body, secret string) string {
	methodUpper := strings.ToUpper(method)
	payload := timestamp + methodUpper + requestPath + body
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func formatAmount(amount string, precision int) (string, error) {
	// Convert string to float64
	floatAmount, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		return "", fmt.Errorf("invalid amount: %v", err)
	}

	// Truncate to desired precision
	power := math.Pow(10, float64(precision))
	truncated := math.Floor(floatAmount*power) / power

	// Format with dynamic precision
	format := fmt.Sprintf("%%.%df", precision)
	return fmt.Sprintf(format, truncated), nil
}

func PrepareRequest(method, path, body, secret string) (*http.Request, error) {
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
	signature := generateSignature(timestamp, method, path, body, apiSecret)
	req, _ := http.NewRequest(method, baseURL+path, nil)
	req.Header.Set("ACCESS-KEY", apiKey)
	req.Header.Set("ACCESS-SIGN", signature)
	req.Header.Set("ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("ACCESS-PASSPHRASE", apiPassphrase)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("locale", "en_US")
	return req, nil
}

func DoRequest(req *http.Request) (map[string]any, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var result map[string]any
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return result, nil
}

func getBalance() ([]map[string]any, error) {
	req, err := PrepareRequest(GET, Assets, "", apiSecret)
	if err != nil {
		return nil, err
	}

	result, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	if result["data"] == nil {
		return nil, fmt.Errorf("API error: %v", result["msg"])
	}

	rawData, ok := result["data"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected data format")
	}
	var assets []map[string]any
	for _, item := range rawData {
		if asset, ok := item.(map[string]any); ok {
			assets = append(assets, asset)
		}
	}
	return assets, nil
}

func placeMarketSell(symbol, amount string) error {
	payload := map[string]string{
		"symbol":    symbol,
		"side":      "sell",
		"orderType": "market",
		"size":      amount,
	}
	bodyBytes, _ := json.Marshal(payload)
	body := string(bodyBytes)

	req, err := PrepareRequest(POST, PlaceOrder, body, apiSecret)
	if err != nil {
		return err
	}
	result, err := DoRequest(req)
	if err != nil {
		return err
	}
	if result["msg"] != "success" {
		return fmt.Errorf("API error: %v", result["msg"])
	}
	return nil
}

func GetSymbolInfo(symbol string) (int, error) {
	req, err := PrepareRequest(GET, fmt.Sprintf("%s?symbol=%s", SymbolInfo, symbol), EmptyBody, apiSecret)
	if err != nil {
		return 0, err
	}

	result, err := DoRequest(req)
	if err != nil {
		return 0, err
	}
	if result["msg"] != "success" {
		return 0, fmt.Errorf("API error: %v", result["msg"])
	}

	dataSlice, ok := result["data"].([]any)
	if !ok || len(dataSlice) == 0 {
		return 0, fmt.Errorf("unexpected data format or empty data")
	}

	firstItem, ok := dataSlice[0].(map[string]any)
	if !ok {
		return 0, fmt.Errorf("unexpected item format in data")
	}

	rawQP, exists := firstItem["quantityPrecision"]
	if !exists {
		return 0, fmt.Errorf("quantityPrecision not found")
	}
	qpStr, ok := rawQP.(string)
	if !ok {
		return 0, fmt.Errorf("invalid quantityPrecision format")

	}

	qpInt, err := strconv.Atoi(qpStr)
	if err != nil {
		return 0, fmt.Errorf("invalid quantityPrecision format")
	}
	return qpInt, nil
}

func GetOrderInfo(symbol string) (map[string]any, error) {
	req, err := PrepareRequest(GET, fmt.Sprintf("%s?symbol=%s", UnfilledOrders, symbol), EmptyBody, apiSecret)
	if err != nil {
		return nil, err
	}

	result, err := DoRequest(req)
	if err != nil {
		return nil, err
	}
	if result["msg"] != "success" {
		return nil, fmt.Errorf("API error: %v", result["msg"])
	}
	dataSlice, ok := result["data"].([]any)
	if !ok || len(dataSlice) == 0 {
		return nil, fmt.Errorf("unexpected data format or empty data")
	}

	firstItem, ok := dataSlice[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected item format in data")
	}

	return firstItem, nil
}

func CheckBalanceCmd() tea.Cmd {

	// async command to fetch balances and return balanceMsg
	fetch := func() tea.Msg {
		assets, err := getBalance()
		var display []string
		if err != nil {

			return balanceMsg{balances: display, err: err, date: time.Now()}
		}
		var symbol string
		for _, asset := range assets {
			coin := asset["coin"].(string)
			available := asset["available"].(string)
			if coin != "USDT" && available != "0" {
				symbol = coin + "USDT"
				precision, err := GetSymbolInfo(symbol)
				if err != nil {
					display = append(display, fmt.Sprintf("%s - %s: %s", available, coin, err))
					continue
				}
				amount, err := formatAmount(available, precision)
				if err != nil {
					display = append(display, fmt.Sprintf("%s - %s: %s", available, amount, err))
					continue
				}

				err = placeMarketSell(symbol, amount)
				if err != nil {
					amount, _ := strconv.ParseFloat(amount, 64)
					adjusted := amount * 0.99
					formattedAmount, err := formatAmount(fmt.Sprintf("%f", adjusted), precision)
					if err != nil {
						display = append(display, fmt.Sprintf("%s - %s: %s", available, coin, err))
						continue
					}
					if adjusted <= 0 {
						display = append(display, fmt.Sprintf("%s - %s: amount too small after adjustment", available, coin))
						continue
					}
					err = placeMarketSell(symbol, formattedAmount)
					if err != nil {
						display = append(display, fmt.Sprintf("%f - %s: tried to sell %s - %s", amount, coin, formattedAmount, err))
					} else {
						display = append(display, fmt.Sprintf("%s - %s", available, coin))
					}
				} else {
					display = append(display, fmt.Sprintf("%s - %s", available, coin))
				}
			}
		}
		return balanceMsg{balances: display, err: err, date: time.Now()}
	}

	return fetch
}
