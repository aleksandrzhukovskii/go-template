package fiber

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/aleksandrzhukovskii/go-template/internal/model"
)

func (s *Service) Main(c *fiber.Ctx) error {
	body := c.Body()

	var response bytes.Buffer

	response.WriteString(fmt.Sprintf("Method: %s\nURL: %s\nRemote Addr: %s\nHeaders:\n", c.Method(),
		c.OriginalURL(), c.IP()))
	for name, values := range c.GetReqHeaders() {
		for _, value := range values {
			response.WriteString(fmt.Sprintf("\t%s: %s\n", name, value))
		}
	}

	if len(body) > 0 {
		response.WriteString(fmt.Sprintf("Body: %s\n", string(body)))
	}

	return c.Status(fiber.StatusOK).Send(response.Bytes())
}

func (s *Service) AddProduct(c *fiber.Ctx) error {
	id, err := s.db.Add(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(&model.Product{ID: id})
}

func (s *Service) UpdateProduct(c *fiber.Ctx) error {
	prod, err := model.ParseProduct(c.FormValue("id"), c.FormValue("name"), c.FormValue("price"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	err = s.db.Update(c.Context(), prod)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsUpdated) {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"msg": "Product updated"})
}

func (s *Service) DeleteProduct(c *fiber.Ctx) error {
	id := c.FormValue("id")
	if err := s.db.Delete(c.Context(), id); err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, model.ErrorNoRowsDeleted) {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"msg": "Product deleted"})
}

func (s *Service) GetProduct(c *fiber.Ctx) error {
	id := c.FormValue("id")
	val, err := s.db.Get(c.Context(), id)
	if err != nil {
		status := fiber.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = fiber.StatusBadRequest
		}
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(&val)
}

func (s *Service) GetProducts(c *fiber.Ctx) error {
	val, err := s.db.GetAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(&val)
}
