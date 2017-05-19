package resources

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Financial-Times/concept-search-api/service"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockConceptSearchService struct {
	mock.Mock
}

func (s *mockConceptSearchService) FindAllConceptsByType(conceptType string) ([]service.Concept, error) {
	args := s.Called(conceptType)
	return args.Get(0).([]service.Concept), args.Error(1)
}

func dummyGenres() []service.Concept {
	return []service.Concept{
		service.Concept{
			Id:          "http://api.ft.com/things/1",
			ApiUrl:      "http://api.ft.com/things/1",
			PrefLabel:   "Test Genre 1",
			ConceptType: "http://www.ft.com/ontology/Genre",
		},
		service.Concept{
			Id:          "http://api.ft.com/things/2",
			ApiUrl:      "http://api.ft.com/things/2",
			PrefLabel:   "Test Genre 2",
			ConceptType: "http://www.ft.com/ontology/Genre",
		},
	}
}

func TestConceptSearchByType(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=http%3A%2F%2Fwww.ft.com%2Fontology%2FGenre", nil)

	genres := dummyGenres()
	svc := mockConceptSearchService{}
	svc.On("FindAllConceptsByType", "http://www.ft.com/ontology/Genre").Return(genres, nil)
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusOK, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string][]service.Concept)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Len(t, respObject["concepts"], 2, "concepts")
	assert.True(t, reflect.DeepEqual(respObject["concepts"], genres))
}

func TestConceptSearchByTypeClientError(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=http%3A%2F%2Fwww.ft.com%2Fontology%2FFoo", nil)

	svc := mockConceptSearchService{}
	svc.On("FindAllConceptsByType", mock.AnythingOfType("string")).Return([]service.Concept{}, service.ErrInvalidConceptType)
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusBadRequest, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, service.ErrInvalidConceptType.Error(), respObject["message"], "error message")
}

func TestConceptSearchByTypeServerError(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=http%3A%2F%2Fwww.ft.com%2Fontology%2FGenre", nil)

	expectedError := errors.New("Test error")
	svc := mockConceptSearchService{}
	svc.On("FindAllConceptsByType", mock.AnythingOfType("string")).Return([]service.Concept{}, expectedError)
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusInternalServerError, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, expectedError.Error(), respObject["message"], "error message")
}

func TestConceptSeachByTypeNoType(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts", nil)

	svc := mockConceptSearchService{}
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusBadRequest, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, service.ErrInvalidConceptType.Error(), respObject["message"], "error message")
	svc.AssertExpectations(t)
}

func TestConceptSeachByTypeBlankType(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=", nil)

	svc := mockConceptSearchService{}
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusBadRequest, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, service.ErrInvalidConceptType.Error(), respObject["message"], "error message")

	svc.AssertExpectations(t)
}

func TestConceptSeachByTypeMultipleTypes(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=http%3A%2F%2Fwww.ft.com%2Fontology%2Fperson%2FPerson&type=http%3A%2F%2Fwww.ft.com%2Fontology%2FGenre", nil)

	svc := mockConceptSearchService{}
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusBadRequest, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, service.ErrInvalidConceptType.Error(), respObject["message"], "error message")
	svc.AssertExpectations(t)
}

func TestConceptSeachByTypeAndValue(t *testing.T) {
	req := httptest.NewRequest("GET", "/concepts?type=http%3A%2F%2Fwww.ft.com%2Fontology%2FGenre&q=fast", nil)

	svc := mockConceptSearchService{}
	endpoint := NewHandler(&svc)

	router := mux.NewRouter()
	router.HandleFunc("/concepts", endpoint.ConceptSearch).Methods("GET")

	actual := httptest.NewRecorder()
	router.ServeHTTP(actual, req)

	assert.Equal(t, http.StatusBadRequest, actual.Code, "http status")
	assert.Equal(t, "application/json", actual.Header().Get("Content-Type"), "content-type")

	respObject := make(map[string]string)
	err := json.Unmarshal(actual.Body.Bytes(), &respObject)
	if err != nil {
		t.Errorf("Unmarshalling request response failed. %v", err)
	}

	assert.Equal(t, service.ErrInvalidConceptType.Error(), respObject["message"], "error message")
	svc.AssertExpectations(t)
}
