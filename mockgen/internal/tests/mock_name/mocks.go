package mock_names

//go:generate mockgen -mock_names=Service=UserServiceMock -package mocks -destination mocks/user_service.go -self_package github.com/canonical/gomock/mockgen/internal/tests/mock_name/mocks github.com/canonical/gomock/mockgen/internal/tests/mock_name/user Service
//go:generate mockgen -mock_names=Service=PostServiceMock -package mocks -destination mocks/post_service.go -self_package github.com/canonical/gomock/mockgen/internal/tests/mock_name/mocks github.com/canonical/gomock/mockgen/internal/tests/mock_name/post Service
