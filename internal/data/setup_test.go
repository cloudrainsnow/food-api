package data

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"testing"

	_ "github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// IMPORTANT -- change the values below to ones that work for your system. The only value you should have to
// worry about is port; if you have something using port 5433, change it to some other value (an unused port)
var (
	host     = "localhost"
	user     = "postgres"
	password = "secret"
	dbName   = "foodapi_test"
	port     = "5433"
	dsn      = "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable timezone=UTC connect_timeout=5"
)

var models Models
var testDB *sql.DB
var resource *dockertest.Resource
var pool *dockertest.Pool

func TestMain(m *testing.M) {
	p, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not connect to docker: %s", err)
	}

	pool = p

	opts := dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14.0",
		Env: []string{
			"POSTGRES_USER=" + user,
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
		},
		ExposedPorts: []string{"5432"},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"5432": {
				{HostIP: "0.0.0.0", HostPort: port},
			},
		},
	}

	resource, err = pool.RunWithOptions(&opts)
	if err != nil {
		log.Fatalf("could not start resource: %s", err)
	}

	if err := pool.Retry(func() error {
		var err error
		testDB, err = sql.Open("pgx", fmt.Sprintf(dsn, host, port, user, password, dbName))
		if err != nil {
			log.Println("Error:", err)
			return err
		}
		return testDB.Ping()
	}); err != nil {
		_ = pool.Purge(resource)
		log.Fatalf("could not connect to docker: %s", err)
	}

	// get our models
	models = New(testDB)

	err = createTables(testDB)
	if err != nil {
		log.Fatalf("could not create tables: %v", err)
	}

	err = insertData(testDB)
	if err != nil {
		log.Fatalf("could not create tables: %v", err)
	}

	code := m.Run()

	if err := pool.Purge(resource); err != nil {
		log.Fatalf("could not purge resource: %s", err)
	}

	os.Exit(code)
}

// createTables will create all the tables in our test database, duplicating the structure
// of the production environment
func createTables(db *sql.DB) error {
	stmt := `
	--
-- Name: countries; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.countries (
    id integer NOT NULL,
    country_name character varying(512),
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: countries_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.countries ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.countries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: foods; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.foods (
    id integer NOT NULL,
    known_as character varying(512),
    country_id integer,
    make_year integer,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    slug character varying(512),
    description text
);


--
-- Name: foods_tastes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.foods_tastes (
    id integer NOT NULL,
    food_id integer,
    taste_id integer,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: foods_tastes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.foods_tastes ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.foods_tastes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: foods_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.foods ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.foods_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: tastes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tastes (
    id integer NOT NULL,
    taste character varying(255),
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: tastes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.tastes ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.tastes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tokens (
    id integer NOT NULL,
    user_id integer,
    email character varying(255) NOT NULL,
    token character varying(255) NOT NULL,
    token_hash bytea NOT NULL,
    expiry timestamp with time zone NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL
);


--
-- Name: tokens_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.tokens ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id integer NOT NULL,
    email character varying(255),
    first_name character varying(255) NOT NULL,
    last_name character varying(255) NOT NULL,
    password character varying(60) NOT NULL,
    created_at timestamp without time zone NOT NULL,
    updated_at timestamp without time zone NOT NULL,
    user_active integer DEFAULT 0
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.users ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);
`

	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}
	return nil
}

// insertData inserts a minimal amount of test data into the test database
func insertData(db *sql.DB) error {

	// insert one country
	stmt := `
	insert into countries (country_name, created_at, updated_at)
	values ('John Smith', '2022-03-05 00:00:01', '2022-03-05 00:00:01')`
	_, err := db.Exec(stmt)
	if err != nil {
		return err
	}

	// insert all tastes
	stmt = `insert into tastes (taste, created_at, updated_at)
	values 
	('umami', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('sweet', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('sour', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('salty', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('bitter', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('spicy', '2022-01-01 01:00:00', '2022-01-01 01:00:00'),
	('oily', '2022-01-01 01:00:00', '2022-01-01 01:00:00')`
	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	// insert one food
	stmt = `
	insert into foods (known_as, country_id, make_year, created_at, updated_at, slug, description)
	values
	('Hamburger', 1, 2022, '2022-01-01 01:00:00', '2022-01-01 01:00:00', 'hamburger', 'Hamburger description')`
	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	// assign a taste to the food
	stmt = `
	insert into foods_tastes (food_id, taste_id, created_at, updated_at)
	values
	(1, 3, '2022-01-01 01:00:00', '2022-01-01 01:00:00')`
	_, err = db.Exec(stmt)
	if err != nil {
		return err
	}

	// you can do the same thing for users & tokens, of course...

	return nil
}
