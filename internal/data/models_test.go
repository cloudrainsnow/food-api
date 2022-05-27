package data

import "testing"

func Test_Ping(t *testing.T) {
	err := testDB.Ping()
	if err != nil {
		t.Error("failed to ping database")
	}
}

func TestFood_GetAll(t *testing.T) {
	all, err := models.Food.GetAll()
	if err != nil {
		t.Error("failed to get all foods", err)
	}

	if len(all) != 1 {
		t.Error("failed to get the correct number of foods")
	}
}

func TestFood_GetOneByID(t *testing.T) {
	f, err := models.Food.GetOneById(1)
	if err != nil {
		t.Error("failed to get one food by id", err)
	}

	if f.KnownAs != "Hamburger" {
		t.Errorf("expected title to be Hamburger but got %s", f.KnownAs)
	}
}

func TestFood_GetOneBySlug(t *testing.T) {
	f, err := models.Food.GetOneBySlug("hamburger")
	if err != nil {
		t.Error("failed to get one food by slug", err)
	}

	if f.KnownAs != "Hamburger" {
		t.Errorf("expected title to be Hamburger but got %s", f.KnownAs)
	}

	_, err = models.Food.GetOneBySlug("bad-slug")
	if err == nil {
		t.Error("did not get an error when attempting to fetch non-existent slug")
	}
}
