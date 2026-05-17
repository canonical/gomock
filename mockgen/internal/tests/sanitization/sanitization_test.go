package sanitization

import (
	"testing"

	"github.com/canonical/gomock/gomock"
	any0 "github.com/canonical/gomock/mockgen/internal/tests/sanitization/any"
	"github.com/canonical/gomock/mockgen/internal/tests/sanitization/mockout"
)

func TestSanitization(t *testing.T) {
	ctrl := gomock.NewController(t)
	m := mockout.NewMockAnyMock(ctrl)
	m.EXPECT().Do(gomock.Any(), 1)
	m.Do(&any0.Any{}, 1)
}
