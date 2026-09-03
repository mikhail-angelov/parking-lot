// parking-bot — Telegram Mini App-обёртка над Parking Puzzle.
//
// Бот открывает игру (GitHub Pages) как Telegram WebApp через кнопку
// «Играть»: и в inline-клавиатуре /start, и в menu button чата.
//
// Запуск:
//
//	TELEGRAM_BOT_TOKEN=<token> go run ./cmd/bot
//
// Нужен токен от @BotFather. Регистрация Mini App и кнопки меню описаны в
// README (секция «Telegram Mini App»); menu button бот ставит сам при старте.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// GameURL — точка входа Mini App (GitHub Pages).
const GameURL = "https://mikhail-angelov.github.io/parking-lot/"

const apiBase = "https://api.telegram.org"

type update struct {
	UpdateID int `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
	CallbackQuery json.RawMessage `json:"callback_query"`
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set (get one from @BotFather)")
	}
	api := apiBase + "/bot" + token

	// Menu button: всегда видимая кнопка «Играть» в чате с ботом.
	if err := setMenuButton(api); err != nil {
		log.Printf("warning: setChatMenuButton: %v", err)
	}

	log.Printf("parking-bot started, game at %s", GameURL)
	ctx, stop := signal.NotifyContext(nil, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	offset := 0
	for {
		select {
		case <-ctx.Done():
			log.Print("stopping")
			return
		default:
		}
		updates, err := getUpdates(api, offset)
		if err != nil {
			log.Printf("getUpdates: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			if u.Message == nil {
				continue
			}
			handleMessage(api, u.Message.Chat.ID, u.Message.Text)
		}
	}
}

func handleMessage(api string, chatID int64, text string) {
	switch {
	case text == "/start" || text == "/play":
		sendStart(api, chatID)
	case text == "/help":
		sendText(api, chatID, "Parking Puzzle — головоломка «освободи выезд».\n\n"+
			"Жми «Играть» (или кнопку меню 🎮) — игра откроется прямо в Telegram.\n"+
			"600 уровней, прогресс хранится локально.")
	case strings.HasPrefix(text, "/"):
		sendText(api, chatID, "Неизвестная команда. Попробуй /start или /help.")
	default:
		sendStart(api, chatID)
	}
}

// sendStart отвечает приветствием с web_app-кнопкой.
func sendStart(api string, chatID int64) {
	kb := fmt.Sprintf(
		`{"inline_keyboard":[[{"text":"🎮 Играть","web_app":{"url":%q}}]]}`,
		GameURL)
	sendText(api, chatID,
		"🧩 *Parking Puzzle*\n\nОсвободи выезд красной машине — двигай остальные по осям.\n\n"+
			"600 уровней: лёгкие → экспертные. Прогресс сохраняется автоматически.",
		kb)
}

func sendText(api string, chatID int64, text string, extra ...string) {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprint(chatID))
	params.Set("text", text)
	params.Set("parse_mode", "Markdown")
	if len(extra) > 0 {
		params.Set("reply_markup", extra[0])
	}
	resp, err := http.PostForm(api+"/sendMessage", params)
	if err != nil {
		log.Printf("sendMessage: %v", err)
		return
	}
	defer resp.Body.Close()
}

func getUpdates(api string, offset int) ([]update, error) {
	params := url.Values{}
	params.Set("offset", fmt.Sprint(offset))
	params.Set("timeout", "25")
	resp, err := http.PostForm(api+"/getUpdates", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		OK      bool     `json:"ok"`
		Result  []update `json:"result"`
		Message string   `json:"description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if !out.OK {
		return nil, fmt.Errorf("telegram error: %s", out.Message)
	}
	return out.Result, nil
}

func setMenuButton(api string) error {
	body := fmt.Sprintf(
		`{"menu_button":{"type":"web_app","text":"🎮 Играть","web_app":{"url":%q}}}`,
		GameURL)
	resp, err := http.Post(api+"/setChatMenuButton", "application/json",
		strings.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var out struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("telegram rejected setChatMenuButton")
	}
	return nil
}
