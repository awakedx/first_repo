package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"testTeleBot/models"

	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var chatGroupID int64 = -1002019950072

func CheckAnswers(bot *telego.Bot, collectionUsers *mongo.Collection, collectionQuestions *mongo.Collection, userId int64) {
	var User models.User
	filter := bson.M{"userid": userId}

	err := collectionUsers.FindOne(context.TODO(), filter).Decode(&User)
	if err != nil {
		log.Fatal(err)
	}

	right := 0
	for i, ans := range User.Answers {
		var Question models.Question
		filter := bson.M{"userid": userId, "id": i}
		err := collectionQuestions.FindOne(context.TODO(), filter).Decode(&Question)
		if err != nil {
			log.Fatal()
		}

		if ans == Question.Answer {
			right++
		}
	}

	if right == 5 {
		err = bot.ApproveChatJoinRequest(&telego.ApproveChatJoinRequestParams{ChatID: tu.ID(chatGroupID), UserID: userId})

		msg := tu.Message(tu.ID(userId),
			"Вітаю, ви пройшли перевірку.",
		).WithReplyMarkup(tu.ReplyKeyboardRemove().WithRemoveKeyboard())

		bot.SendMessage(msg)

		if err != nil {
			log.Fatal(err)
		}
		filter := bson.M{"userid": userId}
		edit := bson.M{"$set": bson.M{"is_passed": true, "is_passing": false}}
		_, err := collectionUsers.UpdateOne(
			context.TODO(),
			filter,
			edit,
		)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func PushAnswUser(bot *telego.Bot, collectionUsers *mongo.Collection, collectionQuestions *mongo.Collection, userId int64, result *models.User, update *telego.Update) {

	filter := bson.M{"userid": userId}

	var edit primitive.M

	if result.Question_index == 0 {
		edit = bson.M{"$inc": bson.M{"question_index": 1}}
	} else {
		if result.Question_index < 5 {
			edit = bson.M{"$inc": bson.M{"question_index": 1},
				"$push": bson.M{"answers": update.Message.Text},
			}
		} else {
			edit = bson.M{"$push": bson.M{"answers": update.Message.Text}}
		}

	}

	if result.Question_index <= 5 {
		_, err := collectionUsers.UpdateOne(
			context.TODO(),
			filter,
			edit)
		if err != nil {
			log.Fatal(err)
		}
	}

	if result.Question_index == 5 {
		CheckAnswers(bot, collectionUsers, collectionQuestions, userId)
	}

}

func AskQuestion(bot *telego.Bot, userId int64, answ int64, collectionUsers *mongo.Collection, collectionQuestions *mongo.Collection, result *models.User, update *telego.Update) {
	randOper := []string{"+", "-", "/", "*"}
	firstN := rand.Intn(10)
	secondN := rand.Intn(10)
	iRandOper := rand.Intn(4)
	var rightAnswer float64
	switch randOper[iRandOper] {
	case "+":
		rightAnswer = float64(firstN) + float64(secondN)
	case "-":
		rightAnswer = float64(firstN) - float64(secondN)
	case "/":
		if secondN == 0 {
			for {
				secondN = rand.Intn(10)
				if secondN != 0 {
					break
				}
			}
		}
		rightAnswer = float64(firstN) / float64(secondN)
	case "*":
		rightAnswer = float64(firstN) * float64(secondN)
	}
	question := models.Question{

		Question: fmt.Sprintf("Скільки буде %d %s %d?", firstN, randOper[iRandOper], secondN),
		Answer:   fmt.Sprintf("%.1f", rightAnswer),
		UserId:   userId,
		Id:       answ,
	}
	_, err := collectionQuestions.InsertOne(context.TODO(), question)
	if err != nil {
		log.Fatal(err)
	}
	Variants := make([]float64, 0, 4)
	Variants = append(Variants, rightAnswer)
	for i := 0; i < 3; i++ {
		Variants = append(Variants, float64(rand.Intn(10)))
	}
	rand.Shuffle(len(Variants), func(i, j int) {
		Variants[i], Variants[j] = Variants[j], Variants[i]
	})

	keyboard := tu.Keyboard(
		tu.KeyboardRow(
			tu.KeyboardButton(fmt.Sprintf("%.1f", Variants[0])),
			tu.KeyboardButton(fmt.Sprintf("%.1f", Variants[1])),
		),
		tu.KeyboardRow(
			tu.KeyboardButton(fmt.Sprintf("%.1f", Variants[2])),
			tu.KeyboardButton(fmt.Sprintf("%.1f", Variants[3])),
		),
	)

	msg := tu.Message(tu.ID(userId),
		question.Question,
	).WithReplyMarkup(keyboard)

	bot.SendMessage(msg)

	if err != nil {
		log.Fatal(err)
	}

	PushAnswUser(bot, collectionUsers, collectionQuestions, userId, result, update)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	//DATABASE CONNECTION
	DBURI := os.Getenv("MONGOURI")
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(DBURI).SetServerAPIOptions(serverAPI)
	client, err2 := mongo.Connect(context.TODO(), opts)
	defer func() {
		if err2 = client.Disconnect(context.TODO()); err2 != nil {
			panic(err2)
		}
	}()

	if err2 != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//

	var result bson.M
	if err := client.Database("telegramBot").
		RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		panic(err)
	}
	fmt.Println("pinged your deploy,connected to db")

	//collections of DB
	collectionUsers := client.Database("telegramBot").Collection("users")
	collectionQuestions := client.Database("telegramBot").Collection("questions")

	//bot start
	botToken := os.Getenv("TOKEN")
	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	updates, _ := bot.UpdatesViaLongPolling(nil)
	defer bot.StopLongPolling()
	//

	for update := range updates {

		if update.ChatJoinRequest != nil {
			userChatId := update.ChatJoinRequest.UserChatID

			keyboard := tu.Keyboard(
				tu.KeyboardRow(
					tu.KeyboardButton("Розпочати тест"),
				),
			)
			msg := tu.Message(tu.ID(userChatId),
				"Щоб отримати доступ до групи, спершу пройдіть перевірку",
			).WithReplyMarkup(keyboard)

			bot.SendMessage(msg)
		} else if update.Message.LeftChatMember != nil {
			userId := update.Message.From.ID
			filter := bson.M{"userid": userId}

			_, err := collectionUsers.DeleteOne(context.TODO(), filter)
			if err != nil {
				log.Fatal(err)
			}

			_, err = collectionQuestions.DeleteMany(context.TODO(), filter)
			if err != nil {
				log.Fatal(err)
			}

		} else if update.Message != nil && update.Message.LeftChatMember == nil {

			//check if user in db
			userId := update.Message.From.ID
			filter := bson.M{"userid": userId}
			var result models.User

			if err := collectionUsers.FindOne(context.TODO(), filter).Decode(&result); err == mongo.ErrNoDocuments {
				answers := make([]string, 0, 5)
				result := models.User{
					UserId:         userId,
					Is_passing:     true,
					Is_passed:      false,
					Question_index: 0,
					Answers:        answers,
				}
				_, err := collectionUsers.InsertOne(context.TODO(), result)
				AskQuestion(bot, userId, 0, collectionUsers, collectionQuestions, &result, &update)
				if err != nil {
					log.Fatal(err)
				}

			} else if err := collectionUsers.FindOne(context.TODO(), filter).Decode(&result); result.Is_passed == false && result.Is_passing == true {
				if result.Question_index < 5 {
					userId := update.Message.From.ID
					answ := result.Question_index
					AskQuestion(bot, userId, answ, collectionUsers, collectionQuestions, &result, &update)
				} else if result.Question_index == 5 {
					PushAnswUser(bot, collectionUsers, collectionQuestions, userId, &result, &update)
				}
				if err != nil {
					log.Fatal(err)
				}
			}

		}
	}
}
