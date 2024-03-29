package repository

import (
	"errors"
	commentdto "github.com/hotkimho/realworld-api/controller/dto/comment"
	"github.com/hotkimho/realworld-api/models"
	"gorm.io/gorm"
)

type commentRepository struct{}

func NewCommentRepository() *commentRepository {
	return &commentRepository{}
}

func (repo *commentRepository) Create(
	db *gorm.DB,
	requestDTO commentdto.CreateCommentRequestDTO,
	userID, articleID int64,
) (*models.Comment, error) {

	comment := models.Comment{
		Body:      requestDTO.Body,
		UserID:    userID,
		ArticleID: articleID,
	}

	err := db.Model(comment).Create(&comment).Error
	if err != nil {
		return nil, err
	}

	err = db.Model(comment).Association("User").Find(&comment.User)
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (repo *commentRepository) GetByArticle(
	db *gorm.DB,
	articleID int64,
) ([]models.Comment, error) {

	var comments []models.Comment

	err := db.Model(&models.Comment{}).
		Preload("User").
		Find(&comments, "article_id = ?", articleID).
		Error
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (repo *commentRepository) UpdateByID(
	db *gorm.DB,
	requestDTO commentdto.UpdateCommentRequestDTO,
	commentID, ctxUserID, articleID int64,
) (*models.Comment, error) {

	updateData := map[string]interface{}{}

	updateData["body"] = requestDTO.Body

	comment := models.Comment{
		ID:        commentID,
		UserID:    ctxUserID,
		ArticleID: articleID,
	}

	// 먼저 게시글에 댓글이 있는지 확인
	err := db.Model(&models.Comment{}).
		Where(comment).
		First(&comment).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	err = db.Model(&models.Comment{}).
		Preload("User").
		Where(comment).
		Updates(updateData).
		Find(&comment).
		Error
	if err != nil {
		return nil, err
	}

	return &comment, nil
}

// delete
func (repo *commentRepository) DeleteByID(
	db *gorm.DB,
	commentID, ctxUserID, articleID int64,
) error {

	comment := models.Comment{
		ID:        commentID,
		UserID:    ctxUserID,
		ArticleID: articleID,
	}

	// 먼저 게시글에 댓글이 있는지 확인
	err := db.Model(&models.Comment{}).
		Where(comment).
		First(&comment).
		Error
	if err != nil {
		return err
	}

	err = db.Delete(&models.Comment{}, "id = ?", commentID).Error
	if err != nil {
		return err
	}

	return nil
}
