package bot

import (
	"context"
	"fmt"
	"go-price-tracker/internal/service"
	"html"
	"log/slog"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const maxWorkers = 10

type Bot struct {
	api     *tgbotapi.BotAPI
	service service.TrackerService
}

func NewBot(token string, service service.TrackerService) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Bot{
		api:     api,
		service: service,
	}, nil
}

func (b *Bot) Start(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	slog.Info("Telegram бот успешно запущен", "username", b.api.Self.UserName)

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	defer func() {
		b.api.StopReceivingUpdates()
		wg.Wait()
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Остановка Telegram бота...")
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			if update.Message == nil {
				continue
			}

			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func(msg *tgbotapi.Message) {
					defer func() {
						<-sem
						wg.Done()
					}()
					b.HandleUpdate(msg)
				}(update.Message)
			case <-ctx.Done():
				slog.Info("Остановка Telegram бота...")
				return nil
			}
		}
	}
}

func (b *Bot) HandleUpdate(msg *tgbotapi.Message) {
	if !msg.IsCommand() {
		b.sendReply(msg.Chat.ID, "Отправьте команду, например: <code>/add &lt;ссылка&gt; &lt;цена&gt;</code>")
		return
	}

	switch msg.Command() {
	case "start", "help":
		b.handleHelp(msg)
	case "add":
		b.handleAdd(msg)
	case "list":
		b.handleList(msg)
	default:
		b.sendReply(msg.Chat.ID, "Неизвестная команда. Используйте /help для справки.")
	}
}

func (b *Bot) handleList(msg *tgbotapi.Message) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if msg.From == nil {
		return
	}

	items, err := b.service.GetUserList(ctx, msg.From.ID)
	if err != nil {
		slog.Error("Ошибка получения списка", "error", err, "user_id", msg.From.ID)
		b.sendReply(msg.Chat.ID, "❌ Произошла ошибка при загрузке списка.")
		return
	}

	if len(items) == 0 {
		b.sendReply(msg.Chat.ID, "📭 У вас пока нет отслеживаемых товаров.")
		return
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>Ваши подписки:</b>\n\n")

	for i, item := range items {
		fmt.Fprintf(&sb, "%d. <a href=\"%s\">%s</a>\n", i+1, html.EscapeString(item.URL), html.EscapeString(item.Domain))
		fmt.Fprintf(&sb, "   • Текущая цена: <code>%s</code>\n", html.EscapeString(item.CurrentPrice.StringFixed(2)))
		fmt.Fprintf(&sb, "   • Целевая цена: <code>%s</code>\n\n", html.EscapeString(item.TargetPrice.StringFixed(2)))
	}

	b.sendReply(msg.Chat.ID, sb.String())
}

func (b *Bot) handleAdd(msg *tgbotapi.Message) {
	args := strings.Fields(msg.CommandArguments())
	if len(args) < 2 {
		b.sendReply(msg.Chat.ID, "❌ <b>Ошибка:</b> укажите ссылку и целевую цену.\nФормат: <code>/add &lt;url&gt; &lt;цена&gt;</code>")
		return
	}

	rawURL := args[0]
	targetPriceStr := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if msg.From == nil {
		return
	}
	curPrice, err := b.service.TrackProduct(ctx, msg.From.ID, rawURL, targetPriceStr)
	if err != nil {
		slog.Error("Ошибка добавления товара", "error", err, "user_id", msg.From.ID)
		b.sendReply(msg.Chat.ID, fmt.Sprintf("❌ Не удалось добавить товар: %s", html.EscapeString(err.Error())))
		return
	}
	reply := fmt.Sprintf(
		"✅ <b>Товар успешно добавлен!</b>\n\nТекущая цена: <code>%s</code>\nЦелевая цена: <code>%s</code>\n\nЯ пришлю уведомление, когда цена упадет!",
		html.EscapeString(curPrice.StringFixed(2)),
		html.EscapeString(targetPriceStr),
	)
	b.sendReply(msg.Chat.ID, reply)
}

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	text := "👋 <b>Привет! Я трекер цен.</b>\n\n" +
		"Доступные команды:\n" +
		"• <code>/add &lt;url&gt; &lt;целевая_цена&gt;</code> — начать отслеживать товар\n" +
		"• <code>/list</code> — список ваших подписок\n\n" +
		"<b>Пример:</b>\n" +
		"<code>/add https://shop.com/product 1499.90</code>"

	b.sendReply(msg.Chat.ID, text)
}

func (b *Bot) sendReply(chatID int64, text string) {
	msgConfig := tgbotapi.NewMessage(chatID, text)
	msgConfig.ParseMode = tgbotapi.ModeHTML
	msgConfig.DisableWebPagePreview = true

	if _, err := b.api.Send(msgConfig); err != nil {
		slog.Error("Ошибка отправки сообщения в Telegram", "error", err, "chat_id", chatID)
	}
}
