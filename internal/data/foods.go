package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	slugify "github.com/mozillazg/go-slugify"
)

// Food is the definition of a single food
type Food struct {
	ID          int       `json:"id"`
	KnownAs     string    `json:"known_as"`
	CountryID   int       `json:"country_id"`
	MakeYear    int       `json:"make_year"`
	Slug        string    `json:"slug"`
	Country     Country   `json:"country"`
	Description string    `json:"description"`
	Tastes      []Taste   `json:"tastes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	TasteIDs    []int     `json:"taste_ids,omitempty"`
}

// Country is the definition of a single country
type Country struct {
	ID          int       `json:"id"`
	CountryName string    `json:"country_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Taste is the definition of a single taste type
type Taste struct {
	ID        int       `json:"id"`
	Taste     string    `json:"taste"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// tastesForFood returns all tastes for a given food id
func (f *Food) tastesForFood(foodID int) ([]Taste, []int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	// get tastes
	var tastes []Taste
	var tasteIDs []int
	tasteQuery := `select id, taste, created_at, updated_at from tastes where id in (select taste_id 
				from foods_tastes where food_id = $1) order by taste`

	tRows, err := db.QueryContext(ctx, tasteQuery, foodID)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}
	defer tRows.Close()

	var taste Taste
	for tRows.Next() {

		err = tRows.Scan(
			&taste.ID,
			&taste.Taste,
			&taste.CreatedAt,
			&taste.UpdatedAt)
		if err != nil {
			return nil, nil, err
		}
		tastes = append(tastes, taste)
		tasteIDs = append(tasteIDs, taste.ID)
	}

	return tastes, tasteIDs, nil
}

// GetAll returns a slice of all foods
func (f *Food) GetAll() ([]*Food, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select f.id, f.known_as, f.country_id, f.make_year, f.slug, f.description, f.created_at, f.updated_at,
			c.id, c.country_name, c.created_at, c.updated_at
			from foods f
			left join countries c on (f.country_id = c.id)
			order by f.known_as`

	var foods []*Food

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var food Food
		err := rows.Scan(
			&food.ID,
			&food.KnownAs,
			&food.CountryID,
			&food.MakeYear,
			&food.Slug,
			&food.Description,
			&food.CreatedAt,
			&food.UpdatedAt,
			&food.Country.ID,
			&food.Country.CountryName,
			&food.Country.CreatedAt,
			&food.Country.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// get tastes
		tastes, ids, err := f.tastesForFood(food.ID)
		if err != nil {
			return nil, err
		}
		food.Tastes = tastes
		food.TasteIDs = ids

		foods = append(foods, &food)
	}

	return foods, nil
}

// GetAllPaginated returns a slice of all foods, paginated by limit and offset
func (f *Food) GetAllPaginated(page, pageSize int) ([]*Food, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	limit := pageSize
	offset := (page - 1) * pageSize

	query := `select f.id, f.known_as, f.country_id, f.make_year, f.slug, f.description, f.created_at, f.updated_at,
				c.id, c.country_name, c.created_at, c.updated_at
				from foods f
				left join countries c on (f.country_id = c.id)
				order by f.known_as
				limit $1 offset $2`

	var foods []*Food

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var food Food
		err := rows.Scan(
			&food.ID,
			&food.KnownAs,
			&food.CountryID,
			&food.MakeYear,
			&food.Slug,
			&food.Description,
			&food.CreatedAt,
			&food.UpdatedAt,
			&food.Country.ID,
			&food.Country.CountryName,
			&food.Country.CreatedAt,
			&food.Country.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// get tastes
		tastes, ids, err := f.tastesForFood(food.ID)
		if err != nil {
			return nil, err
		}
		food.Tastes = tastes
		food.TasteIDs = ids

		foods = append(foods, &food)
	}

	return foods, nil
}

// GetOneById returns one food by its id
func (f *Food) GetOneById(foodID int) (*Food, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select f.id, f.known_as, f.country_id, f.make_year, f.slug, f.description, f.created_at, f.updated_at,
				c.id, c.country_name, c.created_at, c.updated_at
				from foods f
				left join countries c on (f.country_id = c.id)
				where f.id = $1`

	row := db.QueryRowContext(ctx, query, foodID)

	var food Food

	err := row.Scan(
		&food.ID,
		&food.KnownAs,
		&food.CountryID,
		&food.MakeYear,
		&food.Slug,
		&food.Description,
		&food.CreatedAt,
		&food.UpdatedAt,
		&food.Country.ID,
		&food.Country.CountryName,
		&food.Country.CreatedAt,
		&food.Country.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// get tastes
	tastes, ids, err := f.tastesForFood(food.ID)
	if err != nil {
		return nil, err
	}
	food.Tastes = tastes
	food.TasteIDs = ids

	return &food, nil
}

// GetOneBySlug returns one food by slug
func (f *Food) GetOneBySlug(slug string) (*Food, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select f.id, f.known_as, f.country_id, f.make_year, f.slug, f.description, f.created_at, f.updated_at,
			c.id, c.country_name, c.created_at, c.updated_at
			from foods f
			left join countries c on (f.country_id = c.id)
			where f.slug = $1`

	row := db.QueryRowContext(ctx, query, slug)

	var food Food

	err := row.Scan(
		&food.ID,
		&food.KnownAs,
		&food.CountryID,
		&food.MakeYear,
		&food.Slug,
		&food.Description,
		&food.CreatedAt,
		&food.UpdatedAt,
		&food.Country.ID,
		&food.Country.CountryName,
		&food.Country.CreatedAt,
		&food.Country.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// get tastes
	tastes, ids, err := f.tastesForFood(food.ID)
	if err != nil {
		return nil, err
	}
	food.Tastes = tastes
	food.TasteIDs = ids

	return &food, nil
}

// Insert saves one food to the database
func (f *Food) Insert(food Food) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `insert into foods (known_as, country_id, make_year, slug, description, created_at, updated_at)
			values ($1, $2, $3, $4, $5, $6, $7) returning id`

	var newID int
	err := db.QueryRowContext(ctx, stmt,
		food.KnownAs,
		food.CountryID,
		food.MakeYear,
		slugify.Slugify(food.KnownAs),
		food.Description,
		time.Now(),
		time.Now(),
	).Scan(&newID)
	if err != nil {
		return 0, err
	}

	// update tastes using taste ids
	if len(food.TasteIDs) > 0 {
		stmt = `delete from foods_tastes where food_id = $1`
		_, err := db.ExecContext(ctx, stmt, food.ID)
		if err != nil {
			return newID, fmt.Errorf("food updated, but tastes not: %s", err.Error())
		}

		// add new tastes
		for _, x := range food.TasteIDs {
			stmt = `insert into foods_tastes (food_id, taste_id, created_at, updated_at)
				values ($1, $2, $3, $4)`
			_, err = db.ExecContext(ctx, stmt, newID, x, time.Now(), time.Now())
			if err != nil {
				return newID, fmt.Errorf("food updated, but tastes not: %s", err.Error())
			}
		}
	}

	return newID, nil
}

// Update updates one food in the database
func (f *Food) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `update foods set
		known_as = $1,
		country_id = $2,
		make_year = $3,
        slug = $4,
    	description = $5,
		updated_at = $6
		where id = $7`

	_, err := db.ExecContext(ctx, stmt,
		f.KnownAs,
		f.CountryID,
		f.MakeYear,
		slugify.Slugify(f.KnownAs),
		f.Description,
		time.Now(),
		f.ID)
	if err != nil {
		return err
	}

	// update tastes using taste ids
	if len(f.TasteIDs) > 0 {
		stmt = `delete from foods_tastes where food_id = $1`
		_, err := db.ExecContext(ctx, stmt, f.ID)
		if err != nil {
			return fmt.Errorf("food updated, but tastes not: %s", err.Error())
		}

		// add new tastes
		for _, x := range f.TasteIDs {
			stmt = `insert into foods_tastes (food_id, taste_id, created_at, updated_at)
				values ($1, $2, $3, $4)`
			_, err = db.ExecContext(ctx, stmt, f.ID, x, time.Now(), time.Now())
			if err != nil {
				return fmt.Errorf("food updated, but tastes not: %s", err.Error())
			}
		}
	}

	return nil
}

// DeleteByID deletes a food by id
func (f *Food) DeleteByID(foodID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	stmt := `delete from foods where id = $1`
	_, err := db.ExecContext(ctx, stmt, foodID)
	if err != nil {
		return err
	}
	return nil
}

// All returns a list of all countries
func (c *Country) All() ([]*Country, error) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	query := `select id, country_name, created_at, updated_at  from countries order by country_name`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var countries []*Country

	for rows.Next() {
		var country Country
		err := rows.Scan(&country.ID, &country.CountryName, &country.CreatedAt, &country.UpdatedAt)
		if err != nil {
			return nil, err
		}
		countries = append(countries, &country)
	}
	return countries, nil
}
