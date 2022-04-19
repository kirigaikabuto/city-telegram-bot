package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
)

var (
	telegramBotApiToken = "5326822961:AAGRO1ZpN4JZg9J2VKY6aIiX-QUufuEovUM"
)

var mainMenuKeyBoard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Заявка"),
		tgbotapi.NewKeyboardButton("Мероприятия"),
	),
)

var applicationMenuKeyBoard = tgbotapi.NewReplyKeyboard(
	tgbotapi.NewKeyboardButtonRow(
		tgbotapi.NewKeyboardButton("Создать Заявку"),
		tgbotapi.NewKeyboardButton("Статус Последней Заявки"),
		tgbotapi.NewKeyboardButton("Список Заявок"),
	),
)

//var createApplicationMenuKeyboard = tgbotapi.NewInlineKeyboardMarkup(
//	tgbotapi.NewInlineKeyboardRow(
//		tgbotapi.NewInlineKeyboardButtonData("Имя", "Yerassyl"),
//		tgbotapi.NewInlineKeyboardButtonData("Фамилия", "Тлеугазы"),
//	))

type UserApplication struct {
	Id              string `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Phone           string `json:"phone_number"`
	Patronymic      string `json:"patronymic"`
	ApplicationType string `json:"app_type"`
	Message         string `json:"message"`
	Address         string `json:"address"`
	FileId          string
}

type Result struct {
	FileId       string `json:"file_id"`
	FileUniqueId string `json:"file_unique_id"`
	FilePath     string `json:"file_path"`
}

type TelegramBotGetFileInfoResponse struct {
	Result Result `json:"result"`
}

func SendApplicationToApi(u *UserApplication) error {
	urlGetInfoAboutFile := "https://api.telegram.org/bot%s/getFile?file_id=%s"
	urlGetInfoAboutFile = fmt.Sprintf(urlGetInfoAboutFile, telegramBotApiToken, u.FileId)
	clt := &http.Client{}
	req, err := http.NewRequest("GET", urlGetInfoAboutFile, nil)
	if err != nil {
		return err
	}
	res, err := clt.Do(req)
	if err != nil {
		return err
	}
	dataBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	resGetInfo := &TelegramBotGetFileInfoResponse{}
	err = json.Unmarshal(dataBytes, &resGetInfo)
	if err != nil {
		return err
	}
	log.Println("from send app info", resGetInfo)
	urlDownloadFile := "https://api.telegram.org/file/bot%s/%s"
	urlDownloadFile = fmt.Sprintf(urlDownloadFile, telegramBotApiToken, resGetInfo.Result.FilePath)
	err = DownloadFile(resGetInfo.Result.FilePath, urlDownloadFile)
	if err != nil {
		return err
	}
	//create application
	urlCreateApp := "http://localhost:8080/application"
	jsonBody, err := json.Marshal(u)
	if err != nil {
		return err
	}
	jsonBodyReader := bytes.NewReader(jsonBody)
	reqPost, err := http.NewRequest("POST", urlCreateApp, jsonBodyReader)
	if err != nil {
		return err
	}
	resPost, err := clt.Do(reqPost)
	if err != nil {
		return err
	}
	dataBytesPost, err := ioutil.ReadAll(resPost.Body)
	if err != nil {
		return err
	}
	resCreateApp := &UserApplication{}
	err = json.Unmarshal(dataBytesPost, &resCreateApp)
	if err != nil {
		return err
	}
	//send photo
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	sendAppUrl := "http://localhost:8080/application/file?id=%s"
	sendAppUrl = fmt.Sprintf(sendAppUrl, resCreateApp.Id)
	fw, err := writer.CreateFormFile("file", resGetInfo.Result.FilePath)
	if err != nil {
		return err
	}
	file, err := os.Open(resGetInfo.Result.FilePath)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, file)
	if err != nil {
		return err
	}
	writer.Close()
	req, err = http.NewRequest("PUT", sendAppUrl, bytes.NewReader(body.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Api-Key", "de05331f-63d1-4141-bb4a-d9abbd82ead8")
	_, err = clt.Do(req)
	if err != nil {
		return err
	}
	return nil
}

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	bot, err := tgbotapi.NewBotAPI(telegramBotApiToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore non-Message updates
			continue
		}
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		switch strings.ToLower(update.Message.Text) {
		case "начать":
			msg.ReplyMarkup = mainMenuKeyBoard
		case "закрыть":
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		case "заявка":
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			msg.ReplyMarkup = applicationMenuKeyBoard
		case "создать заявку":
			temp := &UserApplication{}
			msg.Text = "Напиши ваш номер телефона"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.Phone = u.Message.Text
					break
				}
			}
			msg.Text = "Напиши ваше имя"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.FirstName = u.Message.Text
					break
				}
			}
			msg.Text = "Напиши вашу фамилию"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.LastName = u.Message.Text
					break
				}
			}
			msg.Text = "Напиши ваше отчество"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.Patronymic = u.Message.Text
					break
				}
			}
			msg.Text = "Напиши тип заявки"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.ApplicationType = u.Message.Text
					break
				}
			}
			msg.Text = "Напишите адресс проишествия"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.Address = u.Message.Text
					break
				}
			}
			msg.Text = "Загрузите файл"
			if _, err := bot.Send(msg); err != nil {
				log.Panic(err)
			}
			for u := range updates {
				if u.Message != nil {
					temp.FileId = u.Message.Photo[0].FileID
					break
				}
			}
			msg.Text = "Спасибо за заявку"
			err = SendApplicationToApi(temp)
			if err != nil {
				log.Fatal("error from send application", err)
			}
			log.Println(temp)
		}
		if _, err := bot.Send(msg); err != nil {
			log.Panic(err)
		}
	}
}
