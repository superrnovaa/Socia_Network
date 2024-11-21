package sqlite

import (
	"golang.org/x/crypto/bcrypt"

	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func ConnectDatabase() (*sql.DB, error) {
	var err error
	DB, err = sql.Open("sqlite3", "../../pkg/db/app.db")
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Enable foreign key support
	_, err = DB.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		return nil, fmt.Errorf("error enabling foreign key support: %v", err)
	}

	return DB, nil
}

func ApplyMigrations(db *sql.DB) error {
	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("could not create driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://../../pkg/db/migrations",
		"sqlite3", driver)
	if err != nil {
		return fmt.Errorf("could not create migration instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("could not apply migrations: %v", err)
	}

	// if the database is empty, insert fake data
	if err := insertFakeData(db); err != nil {
		return fmt.Errorf("could not insert fake data: %v", err)
	}

	return nil
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}
func insertFakeData(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Insert users
		users := []struct {
			Username    string
			Nickname    string
			Email       string
			Password    string
			FirstName   string
			AvatarURL   string
			LastName    string
			DateOfBirth string
			AboutMe     string
			IsPublic    bool
		}{
			{"jsmith", "johnny", "jsmith@gmail.com", "jsmith", "John", "ProfileImage.png", "Smith", "2003-05-15", "Tech enthusiast", true},
			{"emjohnson", "emma", "emjohnson@gmail.com", "emjohnson", "Emma", "ProfileImage.png", "Johnson", "2002-08-22", "Bookworm", false},
			{"mwilliams", "mike", "mwilliams@gmail.com", "mwilliams", "Michael", "ProfileImage.png", "Williams", "2003-11-30", "Sports fan", true},
			{"sbrown", "sarah", "sbrown@gmail.com", "sbrown", "Sarah", "ProfileImage.png", "Brown", "2001-02-14", "Art lover", false},
			{"djones", "david", "djones@gmail.com", "djones", "David", "ProfileImage.png", "Jones", "2000-07-01", "Music addict", true},
			{"lgarcia", "lisa", "lgarcia@gmail.com", "lgarcia", "Lisa", "ProfileImage.png", "Garcia", "2000-09-18", "Nature explorer", false},
			{"rmiller", "robert", "rmiller@gmail.com", "rmiller", "Robert", "ProfileImage.png", "Miller", "2000-12-25", "Food critic", true},
			{"jtaylor", "jennifer", "jtaylor@gmail.com", "jtaylor", "Jennifer", "ProfileImage.png", "Taylor", "2004-03-03", "Movie buff", false},
			{"bsmith", "brian", "bsmith@gmail.com", "bsmith", "Brian", "ProfileImage.png", "Smith", "1999-01-01", "Gamer", true},
			{"klee", "kathy", "klee@gmail.com", "klee", "Kathy", "ProfileImage.png", "Lee", "1998-12-12", "Traveler", false},
			{"moha001", "moha001", "moha001@gmail.com", "moha001", "Mohamed", "ProfileImage.png", "Abdulla", "1997-06-06", "tech enthusiast", true},
		}

		for _, user := range users {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			_, err = db.Exec(`
				INSERT INTO users (username, nickname, email, password, first_name, last_name, date_of_birth, about_me, is_public, avatar_url)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				user.Username, user.Nickname, user.Email, hashedPassword, user.FirstName, user.LastName, user.DateOfBirth, user.AboutMe, user.IsPublic, user.AvatarURL)
			if err != nil {
				return err
			}
		}

		// Insert groups
		groups := []struct {
			Name        string
			Description string
			CreatorName string
			ImageURL    string
		}{
			{"Tech Enthusiasts", "A group for discussing the latest in technology", "jsmith", "group1.png"},
			{"Book Club", "Share and discuss your favorite books", "emjohnson", "group2.png"},
			{"Fitness Fanatics", "Tips and motivation for staying fit", "mwilliams", "group3.png"},
			{"Art Lovers", "Appreciate and share all forms of art", "sbrown", "group4.png"},
			{"Foodies Unite", "Discuss and share your culinary experiences", "rmiller", "group5.png"},
		}

		for _, group := range groups {
			_, err = db.Exec(`
				INSERT INTO groups (name, description, creator_id, image_url)
				VALUES (?, ?, (SELECT id FROM users WHERE username = ?), ?)`,
				group.Name, group.Description, group.CreatorName, group.ImageURL)
			if err != nil {
				return err
			}
		}

		// Insert group members
		_, err = db.Exec(`
			INSERT INTO group_members (group_id, user_id, inviter_id, status)
			SELECT 
				g.id,
				u.id,
				g.creator_id,
				CASE WHEN ABS(RANDOM()) % 10 < 8 THEN 'accepted' ELSE 'pending' END
			FROM groups g
			CROSS JOIN users u
			WHERE ABS(RANDOM()) % 100 < 70
		`)
		if err != nil {
			return err
		}

		// Insert normal posts
		posts := []struct {
			Username string
			Title    string
			Content  string
			Privacy  string
		}{
			{"jsmith", "The Future of AI", "Artificial Intelligence is rapidly evolving. What are your thoughts on its impact on society?", "public"},
			{"jsmith", "My Favorite Programming Languages", "I've been coding for years, and these are my top picks for programming languages in 2023.", "private"},
			{"emjohnson", "Book Review: 1984 by George Orwell", "Just finished reading this classic. Here are my thoughts on its relevance today.", "public"},
			{"emjohnson", "My Reading List for Summer", "I've compiled a list of must-read books for this summer. What's on your list?", "private"},
			{"mwilliams", "NBA Finals Predictions", "The playoffs are heating up. Here are my predictions for this year's NBA finals.", "public"},
			{"mwilliams", "My Fitness Journey", "I've been on a fitness journey for the past 6 months. Here's what I've learned.", "private"},
			{"sbrown", "The Evolution of Modern Art", "From impressionism to abstract expressionism, let's explore the evolution of modern art.", "public"},
			{"sbrown", "My Latest Painting Project", "I've been working on a new series of paintings. Here's a sneak peek!", "private"},
			{"bsmith", "The Rise of E-Sports", "E-sports are taking the world by storm. What games do you think will dominate?", "public"},
			{"klee", "Traveling on a Budget", "Here are my top tips for traveling without breaking the bank!", "private"},
		}

		for _, post := range posts {
			_, err = db.Exec(`
				INSERT INTO posts (user_id, title, content, privacy)
				VALUES ((SELECT id FROM users WHERE username = ?), ?, ?, ?)`,
				post.Username, post.Title, post.Content, post.Privacy)
			if err != nil {
				return err
			}
		}

		// Insert group posts
		_, err = db.Exec(`
			INSERT INTO posts (user_id, group_id, title, content, privacy)
			SELECT 
				gm.user_id,
				g.id,
				CASE 
					WHEN g.name = 'Tech Enthusiasts' THEN 'New AI breakthrough!'
					WHEN g.name = 'Book Club' THEN 'This month''s book review'
					WHEN g.name = 'Fitness Fanatics' THEN 'My workout routine'
					WHEN g.name = 'Art Lovers' THEN 'Latest exhibition thoughts'
					WHEN g.name = 'Foodies Unite' THEN 'Recipe of the week'
					ELSE 'Group update'
				END,
				CASE 
					WHEN g.name = 'Tech Enthusiasts' THEN 'A new AI model has shown remarkable progress in natural language understanding.'
					WHEN g.name = 'Book Club' THEN 'Our book of the month, "The Midnight Library", offers a unique perspective on life choices.'
					WHEN g.name = 'Fitness Fanatics' THEN 'I''ve been trying HIIT workouts lately. Here''s my experience and some tips.'
					WHEN g.name = 'Art Lovers' THEN 'The new modern art exhibition downtown is a must-see. Here are my thoughts.'
					WHEN g.name = 'Foodies Unite' THEN 'I''ve perfected my homemade pizza recipe. Here''s how to make it!'
					ELSE 'Exciting things happening in our group. Stay tuned for more updates!'
				END,
				'public'
			FROM groups g
			JOIN group_members gm ON g.id = gm.group_id
			WHERE gm.status = 'accepted'
			GROUP BY g.id
			Having gm.user_id = MIN(gm.user_id)
		`)
		if err != nil {
			return err
		}

		// Insert comments
		comments := []struct {
			PostTitle string
			Username  string
			Content   string
		}{
			{"The Future of AI", "emjohnson", "Fascinating thoughts! I'm both excited and cautious about AI's potential."},
			{"Book Review: 1984 by George Orwell", "jsmith", "Great review! This book seems more relevant now than ever."},
			{"NBA Finals Predictions", "bsmith", "Interesting picks! I have a feeling there might be an upset this year."},
			{"The Evolution of Modern Art", "klee", "I love how you've traced the progression. Abstract expressionism is my favorite!"},
			{"The Rise of E-Sports", "mwilliams", "I think MOBAs and battle royales will continue to dominate the scene."},
		}

		for _, comment := range comments {
			_, err = db.Exec(`
				INSERT INTO comments (post_id, user_id, content,file)
				SELECT p.id, u.id, ?, ""
				FROM posts p
				JOIN users u ON u.username = ?
				WHERE p.title = ?
				LIMIT 1`,
				comment.Content, comment.Username, comment.PostTitle)
			if err != nil {
				return err
			}
		}
		/*
			// Insert reactions
			reactions := []struct {
				PostTitle    string
				Username     string
				ReactionType string
			}{
				{"The Future of AI", "emjohnson", "like"},
				{"Book Review: 1984 by George Orwell", "jsmith", "love"},
				{"NBA Finals Predictions", "bsmith", "wow"},
				{"The Evolution of Modern Art", "klee", "like"},
				{"The Rise of E-Sports", "mwilliams", "like"},
			}

			for _, reaction := range reactions {
				_, err = db.Exec(`
					INSERT INTO reactions (post_id, user_id, reaction_type_id)
					SELECT p.id, u.id, rt.id
					FROM posts p
					JOIN users u ON u.username = ?
					JOIN reaction_types rt ON rt.name = ?
					WHERE p.title = ?
					LIMIT 1`,
					reaction.Username, reaction.ReactionType, reaction.PostTitle)
				if err != nil {
					return err
				}
			}
		*/

		// Insert events
		events := []struct {
			GroupName   string
			CreatorName string
			Title       string
			Description string
			EventDate   string
		}{
			{"Tech Enthusiasts", "jsmith", "AI Workshop", "Learn about the latest in AI", "2023-08-15 14:00:00"},
			{"Book Club", "emjohnson", "Monthly Book Discussion", "Discussing 'The Great Gatsby'", "2023-07-20 19:00:00"},
			{"Fitness Fanatics", "mwilliams", "Group Workout Session", "High-intensity interval training", "2023-07-25 18:00:00"},
			{"Art Lovers", "sbrown", "Virtual Gallery Tour", "Exploring modern art exhibitions", "2023-08-05 15:00:00"},
			{"Foodies Unite", "rmiller", "Cooking Class", "Learn to make authentic Italian pasta", "2023-08-10 17:00:00"},
		}

		for _, event := range events {
			_, err = db.Exec(`
				INSERT INTO events (group_id, creator_id, title, description, event_date)
				VALUES (
					(SELECT id FROM groups WHERE name = ?),
					(SELECT id FROM users WHERE username = ?),
					?, ?, ?
				)`,
				event.GroupName, event.CreatorName, event.Title, event.Description, event.EventDate)
			if err != nil {
				return err
			}
		}

		// Insert event responses
		_, err = db.Exec(`
			INSERT INTO event_responses (event_id, user_id, response)
			SELECT e.id, u.id, CASE WHEN ABS(RANDOM()) % 2 = 0 THEN 'going' ELSE 'not_going' END
			FROM events e
			CROSS JOIN users u
			WHERE ABS(RANDOM()) % 100 < 60
			LIMIT (SELECT COUNT(*) FROM events) * 5
		`)
		if err != nil {
			return err
		}

		// Insert followers
		_, err = db.Exec(`
			INSERT INTO followers (follower_id, followed_id, status)
			SELECT u1.id, u2.id, 'accepted'
			FROM users u1
			CROSS JOIN users u2
			WHERE u1.id != u2.id
			AND ABS(RANDOM()) % 100 < 50
			LIMIT (SELECT COUNT(*) FROM users) * 4
		`)
		if err != nil {
			return err
		}

		// Insert messages
		_, err = db.Exec(`
			INSERT INTO messages (sender_id, receiver_id, content)
			SELECT u1.id, u2.id, 
				CASE 
					WHEN ABS(RANDOM()) % 5 = 0 THEN 'Hey, how are you?'
					WHEN ABS(RANDOM()) % 5 = 1 THEN 'Did you see the latest post?'
					WHEN ABS(RANDOM()) % 5 = 2 THEN 'Looking forward to the next event!'
					WHEN ABS(RANDOM()) % 5 = 3 THEN 'Thanks for the friend request!'
					ELSE 'Have a great day!'
				END
			FROM users u1
			CROSS JOIN users u2
			WHERE u1.id != u2.id
			AND ABS(RANDOM()) % 100 < 30
			LIMIT (SELECT COUNT(*) FROM users) * 3
		`)
		if err != nil {
			return err
		}

		// Insert group messages
		_, err = db.Exec(`
			INSERT INTO messages (sender_id, group_id, content)
			SELECT 
				gm.user_id,
				g.id,
				CASE 
					WHEN g.name = 'Tech Enthusiasts' THEN 'Check out this new tech article!'
					WHEN g.name = 'Book Club' THEN 'What did everyone think of chapter 5?'
					WHEN g.name = 'Fitness Fanatics' THEN 'Great workout today, team!'
					WHEN g.name = 'Art Lovers' THEN 'I found this amazing art installation downtown.'
					WHEN g.name = 'Foodies Unite' THEN 'Has anyone tried that new restaurant on Main St?'
					ELSE 'Hello everyone!'
				END
			FROM groups g
			JOIN group_members gm ON g.id = gm.group_id
			WHERE gm.status = 'accepted'
			GROUP BY g.id
			Having gm.user_id = MIN(gm.user_id)
		`)
		if err != nil {
			return err
		}

		// Insert post viewers
		_, err = db.Exec(`
			INSERT INTO post_viewers (post_id, viewer_id)
			SELECT p.id, u.id
			FROM posts p
			CROSS JOIN users u
			WHERE p.privacy = 'public' OR (p.privacy = 'almost_private' AND u.id != p.user_id)
			AND ABS(RANDOM()) % 100 < 70
			LIMIT (SELECT COUNT(*) FROM posts) * 5
		`)
		if err != nil {
			return err
		}

		// Insert more comments
		_, err = db.Exec(`
			INSERT INTO comments (post_id, user_id, content, file)
			SELECT 
				p.id,
				u.id,
				CASE 
					WHEN ABS(RANDOM()) % 10 = 0 THEN 'Great post! Thanks for sharing.'
					WHEN ABS(RANDOM()) % 10 = 1 THEN 'I completely agree with your points.'
					WHEN ABS(RANDOM()) % 10 = 2 THEN 'This is really interesting. Can you elaborate more?'
					WHEN ABS(RANDOM()) % 10 = 3 THEN 'I have a different perspective on this.'
					WHEN ABS(RANDOM()) % 10 = 4 THEN 'Thanks for bringing this to our attention!'
					WHEN ABS(RANDOM()) % 10 = 5 THEN 'I learned something new from this post.'
					WHEN ABS(RANDOM()) % 10 = 6 THEN 'This reminds me of a similar experience I had.'
					WHEN ABS(RANDOM()) % 10 = 7 THEN 'I''d love to discuss this further.'
					WHEN ABS(RANDOM()) % 10 = 8 THEN 'Can you provide some sources for this information?'
					ELSE 'Keep up the great content!'
				END,
				''  -- Empty string for the file column
			FROM posts p
			CROSS JOIN users u
			WHERE u.id != p.user_id
			AND ABS(RANDOM()) % 100 < 80  -- Increased probability
			LIMIT (SELECT COUNT(*) FROM posts) * 10  -- Increased limit
		`)
		if err != nil {
			return err
		}

		// Insert more reactions
		_, err = db.Exec(`
			INSERT OR IGNORE INTO reactions (post_id, user_id, reaction_type_id)
			SELECT 
				p.id,
				u.id,
				(SELECT id FROM reaction_types ORDER BY RANDOM() LIMIT 1)
			FROM posts p
			CROSS JOIN users u
			WHERE u.id != p.user_id
			AND ABS(RANDOM()) % 100 < 90  -- Increased probability
			LIMIT (SELECT COUNT(*) FROM posts) * 20  -- Increased limit
		`)
		if err != nil {
			return err
		}

		// Insert reactions for comments
		_, err = db.Exec(`
			INSERT OR IGNORE INTO reactions (comment_id, user_id, reaction_type_id)
			SELECT 
				c.id,
				u.id,
				(SELECT id FROM reaction_types ORDER BY RANDOM() LIMIT 1)
			FROM comments c
			CROSS JOIN users u
			WHERE u.id != c.user_id
			AND ABS(RANDOM()) % 100 < 70  -- High probability for comment reactions
			LIMIT (SELECT COUNT(*) FROM comments) * 5  -- Multiple reactions per comment
		`)
		if err != nil {
			return err
		}
	}

	return nil
}
