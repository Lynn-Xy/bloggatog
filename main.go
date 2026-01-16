package main

import (
	cfg "github.com/Lynn-Xy/bloggatog/internal/config"
	"fmt"
	"os"
	"errors"
	"context"
	"io"
	"html"
	"net/http"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/Lynn-Xy/bloggatog/internal/database"
	"log"
	"time"
	"encoding/xml"
	"strconv"
)

type state struct {
	cfg *cfg.Config
	db *database.Queries
}

type command struct {
	Name string
	Arguments []string
}

type commands struct {
	list map[string]func(*state, command) error
}

type RSSItem struct {
	Title string `xml:"title"`
	Link string `xml:"link"`
	Description string `xml:"description"`
	PubDate string `xml:"pubDate"`
}

type RSSFeed struct {
	Channel struct {
		Title string `xml:"title"`
		Link string `xml:"link"`
		Description string `xml:"description"`
		Item []RSSItem `xml:"item"`
	} `xml:"channel"`
}

func (c *commands) Run(s *state, cmd command) error {
	handler, ok := c.list[cmd.Name]
	if !ok {
		return fmt.Errorf("error running cmd: %v", cmd.Name)
	}
	err := handler(s, cmd)
	if err != nil {
		return fmt.Errorf("error executing command %v: %v", cmd.Name, err)
	}
	return nil
}

func (c *commands) Register(name string, f func(*state, command) error) {
	c.list[name] = f
}

func HandlerLogin(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error logging in: current username not found")
}
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error retrieving username from database: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting username: %v", err)
	}
	fmt.Printf("current user: %v is logged in\n", cmd.Arguments[0])
	return nil
}

func HandlerRegister(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error registering user: no username provided")
	}
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err == nil {
		return fmt.Errorf("error registering new user: %v - user already exists in database", cmd.Arguments[0])
	}
	params := database.CreateUserParams{uuid.New(), time.Now(), time.Now(), cmd.Arguments[0]}
	_, err = s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error creating new user: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting current user: %v", err)
	}
	fmt.Printf("current user: %v has been registered\n", cmd.Arguments[0])
	return nil
}

func HandlerAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.Arguments) > 2 {
		return errors.New("Feed name must be only one word, camelcase combinations are permitted")
	}
	feedParams := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: sql.NullString{
			String: cmd.Arguments[0],
			Valid: true,
		},
		Url: cmd.Arguments[1],
		UserID: user.ID,
	}
	feed, err := s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return fmt.Errorf("error creating feed in database: %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	_, err = s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return fmt.Errorf("error creating feed_follow in database: %v", err)
	}
	fmt.Printf(`
	* Id: %v
	* CreatedAt: %v
	* UpdatedAt: %v
	* Name: %v
	* Url: %v
	* UserId: %v`,
	feed.ID,
	feed.CreatedAt,
	feed.UpdatedAt,
	feed.Name,
	feed.Url,
	feed.UserID)
	return nil
}

func HandlerUsers(s *state, cmd command) error {
	users, err := s.db.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving users from database: %v", err)
	}
	for _, user := range users {
		if user.Name == s.cfg.CurrentUsername {
			fmt.Printf("* %v (current)\n", user.Name)
		} else {
			fmt.Printf("* %v\n", user.Name)
		}
	}
	return nil
}

func HandlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error deleting users from database: %v", err)
	}
	fmt.Print("all users deleted from database\n")
	return nil
}

func HandlerAgg(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("time string must be provided")
	}
	time_between_reqs, err := time.ParseDuration(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error parsing time string argument: %v", err)
	}
	fmt.Printf("Collecting feeds every %v", time_between_reqs)
	tick := time.NewTicker(time_between_reqs)
	for ; ; <-tick.C {
		err := scrapeFeeds(s)
		if err != nil {
			log.Printf("error scraping feeds: %v", err)
		}
	}
	return nil
}

func HandlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving feeds from database: %v", err)
	}
	for _, feed := range feeds {
		fmt.Printf("* Name: %v", feed.Name)
		fmt.Printf("* Url: %v", feed.Url)
		user, err := s.db.GetUserNameById(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("error retrieving user name from database: %v", err)
		}
		fmt.Printf("* Username: %v", user)
	}
	return nil
}

func HandlerFollow(s *state, cmd command, user database.User) error {
	url := cmd.Arguments[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error retrieving feed from database: %v", err)
	}
	feedFollowParams := database.CreateFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	feed_follow_row, err := s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return fmt.Errorf("error creating feed_follow_row: %v", err)
	}
	fmt.Printf("* Feed Name: %v", feed_follow_row.FeedName.String)
	fmt.Printf("* User Name: %v", feed_follow_row.UserName)
	return nil
}

func HandlerFollowing(s *state, cmd command, user database.User) error {
	feed_follows, err := s.db.GetFeedFollowForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("error retrieving feed_follows from database: %v", err)
	}
	for _, follow := range feed_follows {
		feedName, err := s.db.GetFeedNameByFeedID(context.Background(), follow.FeedID)
		if err != nil {
			return fmt.Errorf("error retrieving feed name from database: %v", err)
		}
		fmt.Printf("* Feed Name: %v\n", feedName.String)
	}
	return nil
}

func HandlerUnfollow(s *state, cmd command, user database.User) error {
	url := cmd.Arguments[0]
	feed, err := s.db.GetFeedByUrl(context.Background(), url)
	if err != nil {
		return fmt.Errorf("error retrieving feed from database: %v", err)
	}
	params := database.UnfollowFeedByIDParams{
		UserID: user.ID,
		FeedID: feed.ID,
	}
	err = s.db.UnfollowFeedByID(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error unfollowing feed: %v", err)
	}
	return nil
}

func HandlerBrowse(s *state, cmd command, user database.User) error {
	var limit int32
	if len(cmd.Arguments) < 1 {
		limit = 2
	} else {
		limit = int32(strconv.Atoi(cmd.Arguments[0])
	}
	params :=  database.GetXPostsByUserIDParams{
		ID: user.ID,
		Limit: limit,
	}
	posts, err := s.db.GetXPostsByUserID(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error retrieving posts from database: %v", err)
	}
	for _, post := range posts {
		fmt.Printf("* Post: %+v\n", post)
	}
	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		dbUser, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUsername)
		if err != nil {
			return fmt.Errorf("error retrieving username from database: %v", err)
		}
		return handler(s, cmd, dbUser)
	}
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	newClient := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %v", err)
	}
	req.Header.Set("User-Agent", "gator")
	resp, err := newClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending http get request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error retrieving rss feed contents: server response %v", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http get response body: %v", err)
	}
	var feed RSSFeed
	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling http get request response body data: %v", err)
	}
	for idx, item := range feed.Channel.Item {
		cleanedTitle := html.UnescapeString(item.Title)
		feed.Channel.Item[idx].Title = cleanedTitle
		cleanedDescription := html.UnescapeString(item.Description)
		feed.Channel.Item[idx].Description = cleanedDescription
	}
	cleanedTitle := html.UnescapeString(feed.Channel.Title)
	feed.Channel.Title = cleanedTitle
	cleanedDescription := html.UnescapeString(feed.Channel.Description)
	feed.Channel.Description = cleanedDescription
	return &feed, nil
}

func scrapeFeeds(s *state) error {
	username := s.cfg.CurrentUsername
	if username == "guest" || username == "" {
		return errors.New("no user currently logged in")
	}
	ctx := context.Background()
	user, err := s.db.GetUserByName(ctx, username)
	if err != nil { return fmt.Errorf("error retrieving user from database: %v", err)
	}
	feedRow, err := s.db.GetNextFeedToFetchByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("error retrieving next feed from database: %v", err)
	}
	params := MarkFeedFetchedByIDParams{
		LastFetchedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		ID: feedRow.ID,
	}
	err = s.db.MarkFeedFetchedByID(ctx, params)
	if err != nil {
		return fmt.Errorf("error marking feed fetched: %v", err)
	}
	url := feedRow.Url
	RSSfeed, err := fetchFeed(ctx, url)
	if err != nil {
		return fmt.Errorf("error fetching feed from url: %v - %v", url, err)
	}
	for _, item := range RSSfeed.Channel.Item {
		date, err := time.Parse(DateOnly, item.PubDate)
		if err != nil {
			log.Printf("error parsing publication date: %v", err)
			date = time.Now().UTC()
		}
		postParams := CreatePostParams{
			ID: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title: sql.NullString{
				String: item.Title,
				Valid: true,
			},
			Url: item.Link,
			Description: sql.NullString{
				String: item.Description,
				Valid: true,
			},
			PublishedAt: date,
			FeedID: feedRow.ID,
		}

		err = s.db.CreatePost(ctx, postParams)
		if err != nil && strings.Contains(err.Error(), "unique") == false {
			log.Printf("error creating post in database: %v", err)
		}
	}
	return nil
}

func main() {
	config, err := cfg.Read()
	if err != nil {
		fmt.Printf("error reading config: %v", err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", config.DBURL)
	if err != nil {
		log.Printf("error opening database connection: %v", err)
	}
	dbQ := database.New(db)
	s := &state{&config, dbQ}
	commands := commands{
		list: make(map[string]func(*state, command) error),
	}

	commands.Register("login", HandlerLogin)
	commands.Register("register", HandlerRegister)
	commands.Register("reset", HandlerReset)
	commands.Register("users", HandlerUsers)
	commands.Register("agg", HandlerAgg)
	commands.Register("addfeed", middlewareLoggedIn(HandlerAddFeed))
	commands.Register("feeds", HandlerFeeds)
	commands.Register("follow", middlewareLoggedIn(HandlerFollow))
	commands.Register("following", middlewareLoggedIn(HandlerFollowing))
	commands.Register("unfollow", middlewareLoggedIn(HandlerUnfollow))
	commands.Register("browser", middlewareLoggedIn(HandlerBrowse))

	args := os.Args
	if len(args) < 2 {
		fmt.Println("not enough arguments provided")
		os.Exit(1)
	}

	cmd := command{
		Name:      args[1],
		Arguments: args[2:],
	}

	if err := commands.Run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
