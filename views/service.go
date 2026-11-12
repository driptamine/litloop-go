// Package views records view events and updates the corresponding counters atomically.
package views

import (
	"litloop/backend/albums"
	"litloop/backend/movies"
	"litloop/backend/posts"

	"gorm.io/gorm"
)

func RecordPost(tx *gorm.DB, id, user int64) error {
	var post posts.Post
	if err := tx.First(&post, id).Error; err != nil {
		return err
	}
	if err := tx.Create(&posts.PostView{PostID: id, UserID: user}).Error; err != nil {
		return err
	}
	return tx.Model(&post).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error
}
func RecordMovie(tx *gorm.DB, id, user int64) error {
	var movie movies.Movie
	if err := tx.First(&movie, id).Error; err != nil {
		return err
	}
	if err := tx.Create(&movies.MovieView{MovieID: id, UserID: user}).Error; err != nil {
		return err
	}
	return tx.Model(&movie).UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error
}
func RecordAlbum(tx *gorm.DB, id, user int64) error {
	var album albums.Album
	if err := tx.First(&album, id).Error; err != nil {
		return err
	}
	return tx.Create(&albums.AlbumView{AlbumID: id, UserID: user}).Error
}
