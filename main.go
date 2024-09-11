package main

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Definir un modelo para la tabla de palabras
type Word struct {
	ID                 uint    `gorm:"primaryKey"`
	MainWord           *string `gorm:"unique;not null"`
	BaseLanguageID     uint    `gorm:"not null"` // Cambia el nombre aquí
	TranslateWord      *string `gorm:"null"`
	LearningLanguageID uint    `gorm:"null"` // Cambia el nombre aquí
}

type Language struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	Flag string // Campo para el emoji de la bandera
}

type RequestPayload struct {
	WordsList    []Word `json:"wordsList"`
	DeletedWords []Word `json:"deletedWords"`
}

func main() {
	// Abrir la conexión a la base de datos SQLite
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	// Migrar las estructuras a la base de datos
	db.AutoMigrate(&Language{}, &Word{})

	// Inicializar la tabla Language con datos por defecto si está vacía
	initializeLanguages(db)

	// Crear una nueva instancia de Fiber
	app := fiber.New()

	// Middleware para manejar CORS manualmente
	app.Use(func(c *fiber.Ctx) error {
		// Establecer los encabezados de CORS
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE")
		c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Si la solicitud es un preflight (OPTIONS), responder con un estado 204 No Content
		if c.Method() == "OPTIONS" {
			return c.SendStatus(fiber.StatusNoContent)
		}

		// Continuar con el siguiente middleware o controlador
		return c.Next()
	})

	// Ruta POST para recibir un texto y devolver un array de palabras únicas
	app.Post("/submit", func(c *fiber.Ctx) error {
		type Request struct {
			Text               string `json:"text"`
			BaseLanguageID     uint   `json:"base_language_id"`     // Cambia el nombre aquí
			LearningLanguageID uint   `json:"learning_language_id"` // Cambia el nombre aquí
		}

		var req Request
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No se pudo parsear el cuerpo de la solicitud"})
		}

		// Separar el texto en palabras
		words := strings.Fields(req.Text)

		// Eliminar palabras duplicadas
		uniqueWords := make(map[string]bool)
		for _, word := range words {
			uniqueWords[word] = true
		}

		// Guardar palabras únicas en la base de datos si no existen
		for word := range uniqueWords {
			var count int64
			db.Model(&Word{}).Where("main_word = ? AND base_language_id = ?", word, req.BaseLanguageID).Count(&count)
			if count == 0 {
				wordToSave := word // Creación de una variable local para evitar problemas de referencia
				db.Create(&Word{MainWord: &wordToSave, BaseLanguageID: req.BaseLanguageID, LearningLanguageID: req.LearningLanguageID})
			}
		}

		// Preparar las listas para las palabras con y sin traducción
		var wordsWithTranslate []Word
		var wordsWithoutTranslate []Word

		// Consultar las palabras únicas en la base de datos
		for word := range uniqueWords {
			var foundWord Word
			db.Where("main_word = ? AND base_language_id = ?", word, req.BaseLanguageID).First(&foundWord)
			if foundWord.TranslateWord != nil {
				wordsWithTranslate = append(wordsWithTranslate, foundWord)
			} else {
				wordsWithoutTranslate = append(wordsWithoutTranslate, foundWord)
			}
		}

		// Formatear la respuesta
		return c.JSON(fiber.Map{
			"wordswithtranslate":    wordsWithTranslate,
			"wordswithouttranslate": wordsWithoutTranslate,
		})
	})

	// Ruta POST para recibir y actualizar palabras traducidas
	app.Post("/filledwords", func(c *fiber.Ctx) error {
		var payload RequestPayload
		if err := c.BodyParser(&payload); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No se pudo parsear el cuerpo de la solicitud"})
		}

		// Eliminar las palabras de la base de datos
		for _, word := range payload.DeletedWords {
			db.Delete(&Word{}, word.ID)
		}

		// Actualizar las palabras en la base de datos
		for _, word := range payload.WordsList {
			db.Model(&Word{}).Where("id = ?", word.ID).Updates(Word{
				TranslateWord:      word.TranslateWord,
				LearningLanguageID: word.LearningLanguageID,
			})
		}

		return c.JSON(fiber.Map{"status": "success", "message": "Palabras actualizadas y eliminadas correctamente"})
	})

	// Ruta GET para obtener todas las palabras
	app.Get("/words", func(c *fiber.Ctx) error {
		var words []Word
		db.Find(&words)
		return c.JSON(words)
	})

	app.Get("/languages", func(c *fiber.Ctx) error {
		var languages []Language
		db.Find(&languages)
		return c.JSON(languages)
	})

	// Iniciar el servidor en el puerto 3000
	log.Fatal(app.Listen(":3000"))
}

func initializeLanguages(db *gorm.DB) {
	// Contar el número de registros en la tabla Language
	var count int64
	db.Model(&Language{}).Count(&count)

	// Si la tabla está vacía, agregar los idiomas por defecto
	if count == 0 {
		languages := []Language{
			{Name: "English", Flag: "🇬🇧"},    // Emoji de la bandera del Reino Unido
			{Name: "Spanish", Flag: "🇪🇸"},    // Emoji de la bandera de España
			{Name: "French", Flag: "🇫🇷"},     // Emoji de la bandera de Francia
			{Name: "German", Flag: "🇩🇪"},     // Emoji de la bandera de Alemania
			{Name: "Chinese", Flag: "🇨🇳"},    // Emoji de la bandera de China
			{Name: "Japanese", Flag: "🇯🇵"},   // Emoji de la bandera de Japón
			{Name: "Russian", Flag: "🇷🇺"},    // Emoji de la bandera de Rusia
			{Name: "Portuguese", Flag: "🇵🇹"}, // Emoji de la bandera de Portugal
			{Name: "Italian", Flag: "🇮🇹"},    // Emoji de la bandera de Italia
		}
		db.Create(&languages)
	}
}
