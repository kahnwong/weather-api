package qrcode

import (
	"fmt"
	"log"
	"os"

	owm "github.com/briandowns/openweathermap"
	"github.com/gofiber/fiber/v2"
)

var (
	apiKey = os.Getenv("QRCODE_API_KEY")
)

type TitleResponse struct {
	Name string `json:"name"`
}

type QrcodeRequestItem struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"` // base64
}

func TitleGetController(c *fiber.Ctx) error {

	return c.JSON(TitleResponse{
		Name: "foo",
	})
}

func init() {
	w, err := owm.NewCurrent("C", "en", os.Getenv("OPENWEATHER_API_KEY"))
	if err != nil {
		log.Fatalln(err)
	}

	err = w.CurrentByName("Bangkok")
	if err != nil {
	}

	fmt.Println(w.Timezone)
	fmt.Println(w.Weather[0].Description)
	fmt.Println(w.Rain) // one hour, three hour
}
