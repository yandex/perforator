package mock_names

//go:generate mockgen -mock_names=Service=UserServiceMock -package mocks -typed -destination mocks/user_service.go -self_package go.uber.org/mock/mockgen/internal/tests/mock_name/mocks go.uber.org/mock/mockgen/internal/tests/mock_name/user Service
//go:generate mockgen -mock_names=Service=PostServiceMock -package mocks -typed -destination mocks/post_service.go -self_package go.uber.org/mock/mockgen/internal/tests/mock_name/mocks go.uber.org/mock/mockgen/internal/tests/mock_name/post Service
