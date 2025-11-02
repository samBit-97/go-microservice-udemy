package domain

import (
	"context"
	"fmt"
	"log"
	math "math/rand/v2"
	"ride-sharing/services/driver-service/internal/util"
	"ride-sharing/shared/messaging"
	pb "ride-sharing/shared/proto/driver"
	sharedutil "ride-sharing/shared/util"
	"slices"
	"sync"

	"github.com/mmcloughlin/geohash"
)

type driverService struct {
	drivers []*driverInMap
	mu      sync.Mutex
}

type driverInMap struct {
	Driver *pb.Driver
}

// NewDriverService creates a new driver service instance
func NewDriverService() DriverService {
	return &driverService{
		drivers: make([]*driverInMap, 0),
	}
}

func (s *driverService) RegisterDriver(driverID string, packageSlug string) (*pb.Driver, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	randomIndex := math.IntN(len(PredefinedRoutes))
	randomRoute := PredefinedRoutes[randomIndex]

	randomPlate := util.GenerateRandomPlate()
	randomAvatar := sharedutil.GetRandomAvatar(randomIndex)

	// we can ignore this property for now, but it must be sent to the frontend.
	geohashVal := geohash.Encode(randomRoute[0][0], randomRoute[0][1])

	driver := &pb.Driver{
		Id:             driverID,
		Geohash:        geohashVal,
		Location:       &pb.Location{Latitude: randomRoute[0][0], Longitude: randomRoute[0][1]},
		Name:           "Lando Norris",
		PackageSlug:    packageSlug,
		ProfilePicture: randomAvatar,
		CarPlate:       randomPlate,
	}

	s.drivers = append(s.drivers, &driverInMap{
		Driver: driver,
	})

	return driver, nil
}

func (s *driverService) UnregisterDriver(driverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, driver := range s.drivers {
		if driver.Driver.Id == driverID {
			s.drivers = slices.Delete(s.drivers, i, i+1)
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrDriverNotFound, driverID)
}

func (s *driverService) ProcessTripCreatedEvent(ctx context.Context, tripID, userID string) error {
	log.Printf("Processing trip %s for user %s", tripID, userID)
	return nil
}

func (s *driverService) FindAndNotifyDrivers(ctx context.Context, tripEvent messaging.TripCreatedEvent) (string, error) {
	suitableDrivers := s.findAvailableDrivers(tripEvent.Trip.SelectedFare.PackageSlug)
	log.Printf("found suitable drivers: %v", len(suitableDrivers))

	if len(suitableDrivers) == 0 {
		return "", fmt.Errorf("no suitable drivers found")
	}

	return suitableDrivers[0], nil
}

func (s *driverService) findAvailableDrivers(packageType string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	var matchingDrivers []string

	for _, driver := range s.drivers {
		if driver.Driver.PackageSlug == packageType {
			matchingDrivers = append(matchingDrivers, driver.Driver.Id)
		}
	}

	if len(matchingDrivers) == 0 {
		return []string{}
	}

	return matchingDrivers
}
