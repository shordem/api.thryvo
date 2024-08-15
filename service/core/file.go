package core_service

import (
	"errors"
	"mime/multipart"

	"github.com/shordem/api.thryvo/dto"
	"github.com/shordem/api.thryvo/lib/config"
	"github.com/shordem/api.thryvo/model"
	"github.com/shordem/api.thryvo/repository"
	core_repository "github.com/shordem/api.thryvo/repository/core"
	user_repository "github.com/shordem/api.thryvo/repository/user"
	"gorm.io/gorm"
)

var (
	FileVisibilityPublic  = "public"
	FileVisibilityPrivate = "private"
)

type FileServiceInterface interface {
	UploadFile(fileDto dto.FileDTO, file *multipart.FileHeader) (string, error)
	FindAllFiles(pageable core_repository.FilePageable) ([]dto.FileDTO, repository.Pagination, error)
	GetFile(fileName string) (dto.GetFileDTO, error)
	GetFileInfo(fileName string) (dto.FileDTO, error)
}

type fileService struct {
	fileConfig       config.FileConfigInterface
	fileRepository   core_repository.FileRepositoryInterface
	folderRepository core_repository.FolderRepositoryInterface
	userRepository   user_repository.UserRepositoryInterface
}

func NewFileService(
	fileConfig config.FileConfigInterface,
	fileRepository core_repository.FileRepositoryInterface,
	folderRepository core_repository.FolderRepositoryInterface,
	userRepository user_repository.UserRepositoryInterface,
) FileServiceInterface {
	return &fileService{
		fileConfig:       fileConfig,
		fileRepository:   fileRepository,
		folderRepository: folderRepository,
		userRepository:   userRepository,
	}
}

func (f *fileService) ConvertToDTO(file model.File) dto.FileDTO {
	var fileDto dto.FileDTO

	fileDto.ID = file.ID
	fileDto.OriginalName = file.OriginalName
	fileDto.Key = file.Key
	fileDto.MimeType = file.MimeType
	fileDto.Size = file.Size
	fileDto.Visibility = file.Visibility
	fileDto.CreatedAt = file.CreatedAt
	fileDto.UpdatedAt = file.UpdatedAt
	if file.Folder != nil {
		fileDto.Folder = &dto.FolderDTO{
			Name: file.Folder.Name,
		}
	}

	return fileDto
}

func (f *fileService) ConvertToModel(fileDto dto.FileDTO) model.File {
	var file model.File

	file.ID = fileDto.ID
	file.UserID = fileDto.UserID
	file.FolderID = fileDto.FolderID
	file.OriginalName = fileDto.OriginalName
	file.Key = fileDto.Key
	file.MimeType = fileDto.MimeType
	file.Size = fileDto.Size
	file.Visibility = fileDto.Visibility
	file.CreatedAt = fileDto.CreatedAt
	file.UpdatedAt = fileDto.UpdatedAt
	file.DeletedAt.Time = fileDto.DeletedAt

	return file
}

func (f *fileService) UploadFile(fileDto dto.FileDTO, file *multipart.FileHeader) (string, error) {

	if _, err := f.userRepository.FindUserById(fileDto.UserID); err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", errors.New("user not found")
		}

		return "", err
	}

	if fileDto.FolderID != nil {
		if _, err := f.folderRepository.FindFolderById(*fileDto.FolderID); err != nil {
			if err == gorm.ErrRecordNotFound {
				return "", errors.New("folder not found")
			}

			return "", err
		}
	}

	key, err := f.fileConfig.UploadFile(fileDto.UserID.String(), file)
	if err != nil {
		return "", err
	}

	fileDto.Key = key

	fileModel := f.ConvertToModel(fileDto)

	_, err = f.fileRepository.CreateFile(fileModel)
	if err != nil {
		f.fileConfig.DeleteObject(f.fileConfig.GetObjectPath(fileDto.UserID.String(), key))
		return "", err
	}

	return key, nil
}

func (f *fileService) FindAllFiles(pageable core_repository.FilePageable) ([]dto.FileDTO, repository.Pagination, error) {
	files, pagination, err := f.fileRepository.FindAllFiles(pageable)

	if err != nil {
		return nil, repository.Pagination{}, err
	}

	filesDto := []dto.FileDTO{}
	for _, file := range files {
		fileDto := f.ConvertToDTO(file)
		fileDto.Path = f.fileConfig.GetObjectPath(file.UserID.String(), file.Key)

		filesDto = append(filesDto, fileDto)
	}

	return filesDto, pagination, nil
}

func (f *fileService) GetFile(fileName string) (dto.GetFileDTO, error) {
	return f.fileConfig.GetObject(fileName)
}

func (f *fileService) GetFileInfo(fileName string) (dto.FileDTO, error) {
	file, err := f.fileRepository.FindFileByKeyName(fileName)

	if err != nil {
		return dto.FileDTO{}, err
	}

	fileDto := f.ConvertToDTO(file)
	fileDto.Path = f.fileConfig.GetObjectPath(file.UserID.String(), file.Key)

	return fileDto, nil
}
