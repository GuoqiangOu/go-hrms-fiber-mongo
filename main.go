package main

import (
	"context"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoInstance struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var mg MongoInstance

const dbName = "fiber-hrms"
const mongoURI = "mogodb://localhost:27017/" + dbName

type Employee struct {
	ID     string  `json:"id,omitempty" bson:"_id,omitempty"`
	Name   string  `json:"name"`
	Salary float64 `json:"salary"`
	Age    float64 `json:"age"`
}

func Connect() error {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return err
	}

	db := client.Database(dbName)

	mg = MongoInstance{
		Client: client,
		Db:     db,
	}
	return nil

}

func main() {
	if err := Connect(); err != nil {
		log.Fatal(err)
	}
	app := fiber.New()

	// API to get all employees
	app.Get("/employee", func(c *fiber.Ctx) error {
		query := bson.D{{}}
		// all the employees from the db
		cursor, err := mg.Db.Collection("employees").Find(c.Context(), query)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		var employees []Employee = make([]Employee, 0)
		// decode and store into the employees slice
		if err := cursor.All(c.Context(), &employees); err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return c.JSON(employees)
	})

	// API to add/create an employee
	app.Post("/employee", func(c *fiber.Ctx) error {
		collection := mg.Db.Collection("employees")
		employee := new(Employee)
		// parse the data in the post request into the employee variable
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		employee.ID = ""
		// insert the data
		inertionResult, err := collection.InsertOne(c.Context(), employee)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// get the id from insertion result and form a filter query
		filter := bson.D{{Key: "_id", Value: inertionResult.InsertedID}}
		// this will use the filter to find the record created
		createdRecord := collection.FindOne(c.Context(), filter)
		// init the createdEmployee variable
		createdEmployee := &Employee{}
		// decode and send the found employee
		createdRecord.Decode(createdEmployee)
		return c.Status(201).JSON(createdEmployee)
	})

	// API to update an existing employee
	app.Put("/employee/:id", func(c *fiber.Ctx) error {
		idParam := c.Params("id")

		employeeID, err := primitive.ObjectIDFromHex(idParam)

		if err != nil {
			return c.SendStatus(400)
		}

		employee := new(Employee)
		// parse the employee data from the request body
		if err := c.BodyParser(employee); err != nil {
			return c.Status(400).SendString(err.Error())
		}

		// a query to get the data with id
		query := bson.D{{Key: "_id", Value: employeeID}}
		// a query to update using employee data in the request body
		update := bson.D{
			{Key: "$set",
				Value: bson.D{
					{Key: "name", Value: employee.Name},
					{Key: "age", Value: employee.Age},
					{Key: "salary", Value: employee.Salary},
				},
			},
		}

		// use query to find the data with the id, and update using the update query
		err = mg.Db.Collection("employees").FindOneAndUpdate(c.Context(), query, update).Err()
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return c.SendStatus(404)
			}
			return c.SendStatus(500)
		}

		employee.ID = idParam
		// send the updated employee as response
		return c.Status(200).JSON(employee)
	})

	// API to delete/remove an employee with provided id
	app.Delete("/employee/:id", func(c *fiber.Ctx) error {
		employeeID, err := primitive.ObjectIDFromHex(c.Params("id"))

		if err != nil {
			return c.SendStatus(400)
		}

		query := bson.D{{Key: "_id", Value: employeeID}}
		result, err := mg.Db.Collection("employees").DeleteOne(c.Context(), &query)
		if err != nil {
			return c.SendStatus(500)
		}

		// no record with the provided id
		if result.DeletedCount < 1 {
			return c.SendStatus(404)
		}

		return c.Status(200).JSON("record deleted")
	})

	log.Fatal(app.Listen(":3000"))
}
