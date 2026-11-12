// Package movies contains the movies domain models and SQL relationships.
package model

import (
	users "litloop/backend/users/model"
	"time"
)

type Movie struct {
	ID               int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	TmdbID           *int64 `json:"tmdb_id" gorm:"column:tmdb_id"`
	ImdbID           *int64 `json:"imdb_id" gorm:"column:imdb_id"`
	Title            string `json:"title" gorm:"column:title"`
	Description      string `json:"description" gorm:"column:description"`
	MovieFile        string `json:"movie_file" gorm:"column:movie_file"`
	Poster           string `json:"poster" gorm:"column:poster"`
	LikesCount       int64  `json:"likes_count" gorm:"column:likes_count;default:0"`
	DislikesCount    int64  `json:"dislikes_count" gorm:"column:dislikes_count;default:0"`
	ViewsCount       int64  `json:"views_count" gorm:"column:views_count;default:0"`
	ImpressionsCount int64  `json:"impressions_count" gorm:"column:impressions_count;default:0"`
}

func (Movie) TableName() string { return "movies_movie" }

type ImdbMovie struct {
	ID             int64      `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ImdbID         string     `json:"imdb_id" gorm:"column:imdb_id;uniqueIndex;size:20"`
	Title          string     `json:"title" gorm:"column:title;size:255"`
	OriginalTitle  string     `json:"original_title" gorm:"column:original_title;size:255"`
	URL            string     `json:"url" gorm:"column:url"`
	TitleType      string     `json:"title_type" gorm:"column:title_type;size:50"`
	ImdbRating     *float64   `json:"imdb_rating" gorm:"column:imdb_rating"`
	RuntimeMinutes *int64     `json:"runtime_minutes" gorm:"column:runtime_minutes"`
	Year           *int64     `json:"year" gorm:"column:year"`
	ReleaseDate    *time.Time `json:"release_date" gorm:"column:release_date;type:date"`
	Genres         string     `json:"genres" gorm:"column:genres;size:255"`
	NumVotes       *int64     `json:"num_votes" gorm:"column:num_votes"`
	Directors      string     `json:"directors" gorm:"column:directors;size:255"`
	Description    string     `json:"description" gorm:"column:description"`
	CreatedAt      *time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime:false;autoUpdateTime:false;type:date"`
	ModifiedAt     *time.Time `json:"modified_at" gorm:"column:modified_at;type:date"`
	Position       *int64     `json:"position" gorm:"column:position"`
}

func (ImdbMovie) TableName() string { return "movies_imdbmovie" }

type WatchlistItem struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index;uniqueIndex:uq_movies_watchlistitem"`
	MovieID    int64       `json:"movie_id" gorm:"column:movie_id;index;uniqueIndex:uq_movies_watchlistitem"`
	UserRating *float64    `json:"user_rating" gorm:"column:user_rating"`
	DateAdded  time.Time   `json:"date_added" gorm:"column:date_added;autoCreateTime;type:date"`
	DateRated  *time.Time  `json:"date_rated" gorm:"column:date_rated;type:date"`
	Watched    bool        `json:"watched" gorm:"column:watched;default:false"`
	Notes      string      `json:"notes" gorm:"column:notes"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	MovieIDRef *ImdbMovie  `json:"-" gorm:"belongsTo:true;foreignKey:MovieID;references:ID;constraint:OnDelete:CASCADE"`
}

func (WatchlistItem) TableName() string { return "movies_watchlistitem" }

type MovieImpression struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MovieID    int64       `json:"movie_id" gorm:"column:movie_id;index"`
	UserID     *int64      `json:"user_id" gorm:"column:user_id;index"`
	MovieIDRef *Movie      `json:"-" gorm:"belongsTo:true;foreignKey:MovieID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (MovieImpression) TableName() string { return "movies_movieimpression" }

type MovieView struct {
	ID         int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MovieID    int64       `json:"movie_id" gorm:"column:movie_id;index"`
	UserID     int64       `json:"user_id" gorm:"column:user_id;index"`
	MovieIDRef *Movie      `json:"-" gorm:"belongsTo:true;foreignKey:MovieID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef  *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (MovieView) TableName() string { return "movies_movieview" }

type ImdbMovieRating struct {
	ID             int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID         int64       `json:"user_id" gorm:"column:user_id;index;uniqueIndex:uq_movies_imdbmovierating"`
	ImdbMovieID    int64       `json:"imdb_movie_id" gorm:"column:imdb_movie_id;index;uniqueIndex:uq_movies_imdbmovierating"`
	Rating         float64     `json:"rating" gorm:"column:rating"`
	DateRated      *time.Time  `json:"date_rated" gorm:"column:date_rated;type:date"`
	CreatedAt      time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time   `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	UserIDRef      *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
	ImdbMovieIDRef *ImdbMovie  `json:"-" gorm:"belongsTo:true;foreignKey:ImdbMovieID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ImdbMovieRating) TableName() string { return "movies_imdbmovierating" }
