package main

import (
	"bufio"
	translate "cloud.google.com/go/translate/apiv3"
	"context"
	"flag"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	translatable "google.golang.org/genproto/googleapis/cloud/translate/v3"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type Phrase struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	TR       string             `bson:"tr,omitempty"`
	EN       string             `bson:"en,omitempty"`
	Category string             `bson:"category,omitempty"`
	Faulty   bool               `bson:"faulty,omitempty"`
	Pack     int                `bson:"pack,omitempty"`
}

func main() {

	subOne := flag.NewFlagSet("add", flag.ExitOnError)
	//subOneFlagOne := subOne.String("source", "en", "Source")
	//subOneFlagTwo := subOne.String("target", "tr", "Target")
	//subOneFlagThree := subOne.String("category", "", "Category")

	subTwo := flag.NewFlagSet("start", flag.ExitOnError)
	subTwoFlagOne := subTwo.String("category", "", "Category")
	subTwoFlagTwo := subTwo.Bool("faulty", false, "Faulty")
	subTwoFlagThree := subTwo.Int("pack", 0, "Pack")

	if len(os.Args) < 2 {
		fmt.Println("expected 'add' or 'start' subcommands")
		os.Exit(1)
	}

	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI("mongodb+srv://sinanbabacan:abYkrZhRmMnPrpIJ@cluster0.98fpz.mongodb.net/myFirstDatabase?retryWrites=true&w=majority").
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, _ := mongo.Connect(ctx, clientOptions)

	collection := client.Database("lang").Collection("phrase2")

	switch os.Args[1] {

	case "add":
		subOne.Parse(os.Args[2:])
		//result, _ := translateText("psychic-karma-341315", *subOneFlagOne, *subOneFlagTwo, subOne.Args()[0])
		//fmt.Println(*subOneFlagTwo, ": ", result)

		//fmt.Println("Dodo says: do you want me to add the phrase to the database?")

		//	reader := bufio.NewReader(os.Stdin)
		//	fmt.Print("yes or no (y/n)")

		//	char, _, err := reader.ReadRune()

		//	if err != nil {
		//		fmt.Println(err)
		//	}

		//	if char == 'y' {

		phrase := Phrase{
			EN:       subOne.Args()[0],
			TR:       subOne.Args()[1],
			Category: subOne.Args()[2],
			Faulty:   false,
		}

		oneResult, err := collection.InsertOne(context.Background(), phrase)

		if err != nil {
			panic(err)
		}

		fmt.Println(oneResult.InsertedID)
		//} else if char == 'n' {
		fmt.Println("exited")
		//}

	case "start":
		subTwo.Parse(os.Args[2:])

		category := *subTwoFlagOne
		faulty := *subTwoFlagTwo
		pack := *subTwoFlagThree

		filter := bson.D{}

		if faulty {
			filter = append(filter, bson.E{Key: "faulty", Value: faulty})
		}

		if len(category) > 0 {
			filter = append(filter, bson.E{Key: "category", Value: category})
		}

		if pack > 0 {
			filter = append(filter, bson.E{Key: "pack", Value: pack})
		}

		cur, err := collection.Find(context.Background(), filter)

		if err != nil {
			log.Fatal(err)
		}
		defer cur.Close(context.Background())

		var phrases []Phrase

		for cur.Next(context.Background()) {
			result := Phrase{}
			err := cur.Decode(&result)
			if err != nil {
				log.Fatal(err)
			}

			phrases = append(phrases, result)
		}

		scanner := bufio.NewScanner(os.Stdin)
		rand.Seed(time.Now().UnixNano())

		for {
			if phrases == nil || len(phrases) == 0 {
				fmt.Println("nodata")
				os.Exit(1)
			}

			rand.Seed(time.Now().UnixNano())
			i := rand.Intn(len(phrases))

			fmt.Println(i, ".", phrases[i].TR)
			fmt.Print("--> ")
			scanner.Scan()

			t := strings.Replace(scanner.Text(), "\n", "", -1)

			if t == "" {
				continue
			}

			if t == "update" {

				fmt.Printf("en: %s: ", phrases[i].EN)
				scanner.Scan()
				en := strings.Replace(scanner.Text(), "\n", "", -1)

				fmt.Printf("tr: %s: ", phrases[i].TR)
				scanner.Scan()
				tr := strings.Replace(scanner.Text(), "\n", "", -1)

				fmt.Printf("category: %s: ", phrases[i].Category)
				scanner.Scan()
				category := strings.Replace(scanner.Text(), "\n", "", -1)

				fmt.Printf("pack: %d: ", phrases[i].Pack)
				scanner.Scan()
				pack := strings.Replace(scanner.Text(), "\n", "", -1)

				p, _ := strconv.Atoi(pack)

				f := bson.M{"_id": phrases[i].ID}

				u := bson.M{"$set": bson.M{"en": en, "tr": tr, "category": category, "pack": p}}

				update := collection.FindOneAndUpdate(context.Background(), f, u)

				if update.Err() != nil {
					fmt.Println(update.Err())
				}

				continue
			}

			if t == "exit" {
				break
			}

			s := simplifyText(phrases[i].EN)
			t = simplifyText(t)

			if s == t {
				fmt.Println("OK!")

				if phrases[i].Faulty {

					f := bson.M{"_id": phrases[i].ID}

					u := bson.M{"$set": bson.M{"faulty": false}}

					update := collection.FindOneAndUpdate(context.Background(), f, u)

					if update.Err() != nil {
						fmt.Println(update.Err())
					}

					phrases = RemoveIndex(phrases, i)
				}
			} else {
				if phrases[i].Faulty == false {
					phrases[i].Faulty = true

					f := bson.M{"_id": phrases[i].ID}

					u := bson.M{"$set": phrases[i]}

					update := collection.FindOneAndUpdate(context.Background(), f, u)

					if update.Err() != nil {
						fmt.Println(update.Err())
					}
				}

				fmt.Printf("Correct:	%s\n", phrases[i].EN)
				fmt.Printf("Wrong:		%s\n", t)
			}
		}

	default:
		fmt.Println("expected 'add' or 'start' subcommands")
		os.Exit(1)
	}
}

func RemoveIndex(s []Phrase, index int) []Phrase {
	return append(s[:index], s[index+1:]...)
}

func translateText(projectID string, sourceLang string, targetLang string, text string) (string, error) {

	ctx := context.Background()
	client, err := translate.NewTranslationClient(ctx)

	if err != nil {
		return "", err
	}

	defer client.Close()

	req := &translatable.TranslateTextRequest{
		Parent:             fmt.Sprintf("projects/%s/locations/global", projectID),
		SourceLanguageCode: sourceLang,
		TargetLanguageCode: targetLang,
		MimeType:           "text/plain", // Mime types: "text/plain", "text/html"
		Contents:           []string{text},
	}

	resp, err := client.TranslateText(ctx, req)

	if err != nil {
		return "", err
	}

	a := resp.GetTranslations()[0]

	return a.TranslatedText, nil
}

func simplifyText(text string) string {
	text = strings.ToLower(text)
	text = strings.Trim(text, ".")
	text = strings.Trim(text, "?")
	text = strings.Replace(text, ",", "", -1)
	text = strings.Replace(text, "do not", "don't", -1)
	text = strings.Replace(text, "does not", "doesn't", -1)
	text = strings.Replace(text, "will not", "won't", -1)
	text = strings.Replace(text, "cannot", "can't", -1)
	text = strings.Replace(text, "would not", "wouldn't", -1)
	text = strings.Replace(text, "cannot", "can't", -1)
	text = strings.Replace(text, "that is why", "that's why", -1)
	text = strings.Replace(text, "you are", "you're", -1)
	text = strings.Replace(text, "they are", "they're", -1)
	text = strings.Replace(text, "we are", "we're", -1)
	text = strings.Replace(text, "i am", "i'm", -1)
	text = strings.Replace(text, "i have", "i've", -1)
	text = strings.Replace(text, "it is", "it's", -1)
	text = strings.Replace(text, "he is", "he's", -1)
	text = strings.Replace(text, "she is", "she's", -1)

	return text
}
