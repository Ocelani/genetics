package gago

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestInitialized(t *testing.T) {
	var ga = GA{
		NewGenome: NewVector,
		NPops:     1,
		PopSize:   10,
	}
	if ga.Initialized() {
		t.Error("GA should not yet be initialized")
	}
	ga.Initialize()
	if !ga.Initialized() {
		t.Error("GA should be initialized")
	}
}

func TestValidationSuccess(t *testing.T) {
	var err = ga.Validate()
	if err != nil {
		t.Error("GA parameters are invalid")
	}
}

func TestValidationNewGenome(t *testing.T) {
	var genomeFactory = ga.NewGenome
	ga.NewGenome = nil
	if ga.Validate() == nil {
		t.Error("Nil NewGenome should return an error")
	}
	ga.NewGenome = genomeFactory
}

func TestValidationNPopulations(t *testing.T) {
	var nPops = ga.NPops
	ga.NPops = -1
	if ga.Validate() == nil {
		t.Error("Invalid number of Populations should return an error")
	}
	ga.NPops = nPops
}

func TestValidationNIndividuals(t *testing.T) {
	var popSize = ga.PopSize
	ga.PopSize = -1
	if ga.Validate() == nil {
		t.Error("Invalid number of Individuals should return an error")
	}
	ga.PopSize = popSize
}

func TestValidationModel(t *testing.T) {
	var model = ga.Model
	// Check nil model raises error
	ga.Model = nil
	if ga.Validate() == nil {
		t.Error("Nil Model should return an error")
	}
	// Check invalid model raises error
	ga.Model = ModGenerational{
		Selector: SelTournament{
			NContestants: 3,
		},
		MutRate: -1,
	}
	if ga.Validate() == nil {
		t.Error("Invalid Model should return an error")
	}
	ga.Model = model
}

func TestValidationMigFrequency(t *testing.T) {
	var (
		migrator     = ga.Migrator
		migFrequency = ga.MigFrequency
	)
	ga.Migrator = MigRing{}
	ga.MigFrequency = 0
	if ga.Validate() == nil {
		t.Error("Invalid MigFrequency should return an error")
	}
	ga.Migrator = migrator
	ga.MigFrequency = migFrequency
}

func TestValidationSpeciator(t *testing.T) {
	var speciator = ga.Speciator
	ga.Speciator = SpecFitnessInterval{0}
	if ga.Validate() == nil {
		t.Error("Invalid Speciator should return an error")
	}
	ga.Speciator = speciator
}

func TestApplyWithSpeciator(t *testing.T) {
	var speciator = ga.Speciator
	ga.Speciator = SpecFitnessInterval{4}
	if ga.Evolve() != nil {
		t.Error("Calling Apply with a valid Speciator should not return an error")
	}
	ga.Speciator = speciator
}

func TestRandomNumberGenerators(t *testing.T) {
	for i, pop1 := range ga.Populations {
		for j, pop2 := range ga.Populations {
			if i != j && &pop1.RNG == &pop2.RNG {
				t.Error("Population should not share random number generators")
			}
		}
	}
}

func TestBest(t *testing.T) {
	for _, pop := range ga.Populations {
		for _, indi := range pop.Individuals {
			if ga.HallOfFame[0].Fitness > indi.Fitness {
				t.Error("The current best individual is not the overall best")
			}
		}
	}
}

func TestUpdateHallOfFame(t *testing.T) {
	var (
		testCases = []struct {
			hofIn  Individuals
			indis  Individuals
			hofOut Individuals
		}{
			{
				hofIn: Individuals{
					Individual{Fitness: math.Inf(1)},
				},
				indis: Individuals{
					Individual{Fitness: 0},
				},
				hofOut: Individuals{
					Individual{Fitness: 0},
				},
			},
			{
				hofIn: Individuals{
					Individual{Fitness: 0},
					Individual{Fitness: math.Inf(1)},
				},
				indis: Individuals{
					Individual{Fitness: 1},
				},
				hofOut: Individuals{
					Individual{Fitness: 0},
					Individual{Fitness: 1},
				},
			},
		}
	)
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TC %d", i), func(t *testing.T) {
			updateHallOfFame(tc.hofIn, tc.indis)
			// Compare the obtained hall of fame to the expected one)
			for i, indi := range tc.hofIn {
				if indi.Fitness != tc.hofOut[i].Fitness {
					t.Errorf("Expected %v, got %v", tc.hofOut[i], indi)
				}
			}
		})
	}
}

// TestDuration verifies the sum of the duration of each population is higher
// the actual duration. This is due to the fact that each population runs on a
// separate core.
func TestDuration(t *testing.T) {
	var totalDuration time.Duration
	for _, pop := range ga.Populations {
		totalDuration += pop.Age
	}
	if totalDuration < ga.Age {
		t.Error("Inefficient parallelism")
	}
}

func TestSpeciateEvolveMerge(t *testing.T) {
	var (
		rng       = newRand()
		testCases = []struct {
			pop       Population
			speciator Speciator
			model     Model
			err       error
		}{
			{
				pop: Population{
					ID:  "42",
					RNG: rng,
					Individuals: Individuals{
						Individual{Fitness: 0},
						Individual{Fitness: 1},
						Individual{Fitness: 2},
						Individual{Fitness: 3},
						Individual{Fitness: 4},
					},
				},
				speciator: SpecFitnessInterval{3},
				model:     ModIdentity{},
				err:       nil,
			},
			{
				pop: Population{
					ID:  "42",
					RNG: rng,
					Individuals: Individuals{
						Individual{Fitness: 0},
						Individual{Fitness: 1},
						Individual{Fitness: 2},
					},
				},
				speciator: SpecFitnessInterval{4},
				model:     ModIdentity{},
				err:       errors.New("Invalid speciator"),
			},
			{
				pop: Population{
					ID:  "42",
					RNG: rng,
					Individuals: Individuals{
						Individual{Fitness: 0},
						Individual{Fitness: 1},
						Individual{Fitness: 2},
						Individual{Fitness: 3},
						Individual{Fitness: 4},
					},
				},
				speciator: SpecFitnessInterval{3},
				model: ModGenerational{
					Selector: SelTournament{6},
					MutRate:  0.5,
				},
				err: errors.New("Invalid model"),
			},
		}
	)
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TC %d", i), func(t *testing.T) {
			var err = tc.pop.speciateEvolveMerge(tc.speciator, tc.model)
			if (err == nil) != (tc.err == nil) {
				t.Errorf("Wrong error in test case number %d", i)
			}
			// If there is no error check the individuals are ordered as they were
			// at they were initially
			if err == nil {
				for j, indi := range tc.pop.Individuals {
					if indi.Fitness != float64(j) {
						t.Errorf("Wrong result in test case number %d", i)
					}
				}
			}
		})
	}
}

func TestCallback(t *testing.T) {
	var (
		counter          int
		incrementCounter = func(ga *GA) {
			counter++
		}
	)
	ga.Callback = incrementCounter
	ga.Initialize()
	if counter != 1 {
		t.Error("Counter was not incremented by the callback at initialization")
	}
	ga.Evolve()
	if counter != 2 {
		t.Error("Counter was not incremented by the callback at enhancement")
	}
}

func TestGAEvolveModelRuntimeError(t *testing.T) {
	var model = ga.Model
	ga.Model = ModRuntimeError{}
	// Check invalid model doesn't raise error
	if ga.Validate() != nil {
		t.Errorf("Expected nil, got %s", ga.Validate())
	}
	// Evolve
	var err = ga.Evolve()
	if err == nil {
		t.Error("An error should have been raised")
	}
	ga.Model = model
}

func TestGAEvolveSpeciatorRuntimeError(t *testing.T) {
	var speciator = ga.Speciator
	ga.Speciator = SpecRuntimeError{}
	// Check invalid speciator doesn't raise error
	if ga.Validate() != nil {
		t.Errorf("Expected nil, got %s", ga.Validate())
	}
	// Evolve
	var err = ga.Evolve()
	if err == nil {
		t.Error("An error should have been raised")
	}
	ga.Speciator = speciator
}

func TestGAConsistentResults(t *testing.T) {
	var (
		ga1 = GA{
			NewGenome: NewVector,
			NPops:     2,
			PopSize:   10,
			Model: ModGenerational{
				Selector: SelTournament{
					NContestants: 3,
				},
				MutRate: 0.5,
			},
			RNG: rand.New(rand.NewSource(42)),
		}
		ga2 = GA{
			NewGenome: NewVector,
			NPops:     2,
			PopSize:   10,
			Model: ModGenerational{
				Selector: SelTournament{
					NContestants: 3,
				},
				MutRate: 0.5,
			},
			RNG: rand.New(rand.NewSource(42)),
		}
	)

	// Run the first GA
	ga1.Initialize()
	for i := 0; i < 20; i++ {
		ga1.Evolve()
	}

	// Run the second GA
	ga2.Initialize()
	for i := 0; i < 20; i++ {
		ga2.Evolve()
	}

	// Compare best individuals
	if ga1.HallOfFame[0].Fitness != ga2.HallOfFame[0].Fitness {
		t.Errorf("Expected %f, got %f", ga1.HallOfFame[0].Fitness, ga2.HallOfFame[0].Fitness)
	}

}
