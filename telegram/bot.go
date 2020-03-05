package telegram

import (
	"fmt"
	"github.com/grupokindynos/common/telegram"
	"os"
	"sync"
)

type Bot struct {
	telegramBot telegram.TelegramBot
	cache       map[string][2]int //key: the message for the bot | value: [how many times was called][how many times it was not called]
}

var instance *Bot
var once sync.Once

func GetInstance() *Bot {
	once.Do(func() {
		instance = &Bot{telegramBot: telegram.NewTelegramBot(os.Getenv("TELEGRAM_API_KEY"), os.Getenv("TELEGRAM_CHAT_ID")), cache: make(map[string][2]int)}
	})
	return instance
}

func (b *Bot) SendError(msg string) {
	//msg is in table
	if counters, ok := b.cache[msg]; ok {
		//add to times called
		counters[0] = counters[0] + 1
		//times not called goes to 0
		counters[1] = 0
		if counters[0]%60 == 0 { //send message every 60 repetitions = 1 hour
			b.telegramBot.SendError(msg)
		}
		fmt.Print(msg + ": ")
		fmt.Println(counters)
	} else //its an unseen error
	{
		b.telegramBot.SendError(msg)
		//add new msg to cache
		b.cache[msg] = [2]int{0, 0}

	}
	//every other message was not seen +1
	b.updateCache(msg)
}

func (b *Bot) updateCache(msg string) {
	for key, value := range b.cache {
		if key != msg {
			value[1] = value[1] + 1
		}
		//if it was not seen 10 times, remove from cache
		if value[1] == 10 {
			delete(b.cache, key)
		}
	}
}
