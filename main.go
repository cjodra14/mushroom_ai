package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"reflect"
	"strconv"

	dataframe "github.com/rocketlaunchr/dataframe-go"
	"github.com/rocketlaunchr/dataframe-go/imports"
	"github.com/wcharczuk/go-chart/v2"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/palette"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"

	"gonum.org/v1/gonum/mat"

	deep "github.com/patrikeh/go-deep"
)

type GridXYZ struct {
	x []float64
	y []float64
	z *mat.Dense
}

func (g *GridXYZ) Dims() (c, r int) {
	return len(g.x), len(g.y)
}

func (g *GridXYZ) Z(c, r int) float64 {
	return g.z.At(c, r)
}

func (g *GridXYZ) X(c int) float64 {
	return g.x[c]
}

func (g *GridXYZ) Y(r int) float64 {
	return g.y[r]
}

var floatType = reflect.TypeOf(float64(0))
var stringType = reflect.TypeOf("")

func main() {
	filePath := "./mushrooms.csv"
	ctx := context.Background()

	// Load Dataframe of mushroom csv

	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	df, err := imports.LoadFromCSV(ctx, f, imports.CSVLoadOptions{InferDataTypes: true, NilValue: &[]string{"NA"}[0]})
	if err != nil {
		panic(err)
	}

	startRow := 0
	endRow := 2
	fmt.Println(df.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))

	//Plot pie chart of Poisonous - Edible  data balance

	filterFnPoisonous := dataframe.FilterDataFrameFn(func(vals map[interface{}]interface{}, row, nRows int) (dataframe.FilterAction, error) {
		if vals["class"] == "p" {
			return dataframe.DROP, nil
		}
		return dataframe.KEEP, nil
	})

	posisonusMushrooms, err := dataframe.Filter(ctx, df, filterFnPoisonous)
	if err != nil {
		log.Fatal(err)
	}

	filterFnEdible := dataframe.FilterDataFrameFn(func(vals map[interface{}]interface{}, row, nRows int) (dataframe.FilterAction, error) {
		if vals["class"] == "e" {
			return dataframe.DROP, nil
		}
		return dataframe.KEEP, nil
	})

	edibleMushrooms, err := dataframe.Filter(ctx, df, filterFnEdible)
	if err != nil {
		log.Fatal(err)
	}

	edibles := edibleMushrooms.(*dataframe.DataFrame)
	poisonous := posisonusMushrooms.(*dataframe.DataFrame)

	pie := chart.PieChart{
		Title:  "Poisonous - Edible",
		Width:  512,
		Height: 512,
		Values: []chart.Value{
			{Value: float64(poisonous.NRows()), Label: "Poisonous"},
			{Value: float64(edibles.NRows()), Label: "Edible"},
		},
	}

	mushrooms_chart_file, err := os.Create("mushrooms_pie_chart.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := pie.Render(chart.PNG, mushrooms_chart_file); err != nil {
		log.Fatal(err)
	}

	threshold := float64(df.NRows()) * 0.1

	for i, colName := range df.Names() {
		// Get the column as a Series
		series := df.Series[i]

		// Count the null values in the column
		nulls := 0
		for i := 0; i < series.NRows(); i++ {
			val := series.Value(i)
			if val == nil || val == "" {
				nulls++
			}
		}

		// If the count of null values is more than 10% of the total values
		if float64(nulls) > threshold {
			fmt.Printf("Column '%s' has more than 10%% null values\n", colName)
		}
	}

	for i, colName := range df.Names() {
		// Get the column as a Series
		if colName == "habitat" {
			series := df.Series[i]
			for i := 0; i < series.NRows(); i++ {
				val := series.Value(i).(string)

				if !isPossibleHabitat(val) {
					fmt.Printf("%s is not a possible habitat", val)
				}
			}
		}
	}

	if err := df.RemoveSeries("veil-type"); err != nil {
		log.Fatal(err)
	}

	if err := df.RemoveSeries("gill-color"); err != nil {
		log.Fatal(err)
	}

	if err := df.RemoveSeries("bruises"); err != nil {
		log.Fatal(err)
	}

	if err := df.RemoveSeries("ring-type"); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Dataframe Columns: %v\n", df.Names())

	// Iterate over the columns
	columns := df.Names()
	colsToDelete := []string{}
	for i, colName := range columns {
		if colName == "class" {
			continue
		}

		// Get the column as a Series
		series := df.Series[i]

		// Get the underlying data in its original or native format
		dataType := series.Type()
		fmt.Printf("Column '%s' has data type '%s'\n", colName, dataType)

		if dataType == "string" {
			uniqueValues := map[string]int{}
			for i := 0; i < series.NRows(); i++ {
				rowValue := series.ValueString(i)
				uniqueValues[rowValue] = uniqueValues[rowValue] + 1
			}

			fmt.Printf("Column '%s' has unique values '%v'\n", colName, uniqueValues)
			for uniqueValue := range uniqueValues {
				newSerie := dataframe.NewSeriesInt64(colName+"_"+uniqueValue, nil)
				for i := 0; i < series.NRows(); i++ {
					rowValue := series.ValueString(i)
					if rowValue == uniqueValue {
						newSerie.Insert(i, 1)
					} else {
						newSerie.Insert(i, 0)
					}

				}

				if err := df.AddSeries(newSerie, nil); err != nil {
					log.Fatal(err)
				}

			}

			colsToDelete = append(colsToDelete, colName)

			continue
		}

		newSerie := dataframe.NewSeriesInt64(colName+"_new", nil)
		for i := 0; i < series.NRows(); i++ {
			rowValue := series.Value(i)
			newSerie.Insert(i, rowValue)
		}

		if err := df.AddSeries(newSerie, nil); err != nil {
			log.Fatal(err)
		}

		colsToDelete = append(colsToDelete, colName)
	}

	for _, col := range colsToDelete {
		if err := df.RemoveSeries(col); err != nil {
			log.Fatal(err)
		}
	}

	for i, colName := range df.Names() {
		if colName != "class" {
			continue
		}

		series := df.Series[i]

		serieIterator := series.ValuesIterator()

		classNormalizedSerie := dataframe.NewSeriesInt64(colName+"_normalized", nil)

		for {
			row, vals, _ := serieIterator()
			if row == nil {
				break
			}

			if vals.(string) == "e" {
				classNormalizedSerie.Insert(i, 0)
			} else {
				classNormalizedSerie.Insert(i, 1)
			}
		}
		if err := df.AddSeries(classNormalizedSerie, nil); err != nil {
			log.Fatal(err)
		}

		if err := df.RemoveSeries(colName); err != nil {
			log.Fatal(err)
		}

	}

	fmt.Println(df.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))

	corrMatrix := createCorrelationMatrix(df)
	// fmt.Println(corrMatrix)
	saveCorrelationMatrixHeatmap(corrMatrix, "correlation_heatmap.png")

	/*
		// Correlated variables TODO  (research using gonum/mat)
		classIndex := len(df.Names())-1 // Replace this with the index of the 'class' column in your data.
		correlated := searchCorrelatedVariables(corrMatrix, df.Names(), classIndex)
		fmt.Println("Correlated variables:", correlated)
	*/
	colsToPreserve := map[string]bool{"class_normalized": true, "odor_f": true, "odor_n": true, "gill-spacing_c": true, "gill-spacing_w": true, "gill-size_b": true, "gill-size_n": true, "stalk-root_?": true, "stalk-surface-above-ring_k": true, "stalk-surface-above-ring_s": true, "stalk-surface-below-ring_k": true, "stalk-surface-below-ring_s": true, "spore-print-color_h": true, "spore-print-color_k": true, "spore-print-color_n": true, "spore-print-color_w": true, "population_v": true, "habitat_p": true}

	columnsBeforeDelete := df.Names()
	for _, column := range columnsBeforeDelete {
		_, ok := colsToPreserve[column]

		if !ok {
			if err := df.RemoveSeries(column); err != nil {
				log.Fatal(err)
			}
		}
	}

	fmt.Println(df.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))

	xTrain, yTrain, xTest, yTest := split(df, 0.8, "class_normalized")

	fmt.Println(xTrain.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))
	fmt.Println(yTrain.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))
	fmt.Println(xTest.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))
	fmt.Println(yTest.Table(dataframe.TableOptions{R: &dataframe.Range{Start: &startRow, End: &endRow}}))

}

func isPossibleHabitat(habitat string) bool {
	possibleHabitats := []string{"g", "l", "m", "p", "u", "w", "d"}
	for _, possibleHabitat := range possibleHabitats {
		if habitat == possibleHabitat {
			return true
		}
	}

	return false
}

func getFloat(unk interface{}) (float64, error) {
	switch i := unk.(type) {
	case float64:
		return i, nil
	case float32:
		return float64(i), nil
	case int64:
		return float64(i), nil
	case int32:
		return float64(i), nil
	case int:
		return float64(i), nil
	case uint64:
		return float64(i), nil
	case uint32:
		return float64(i), nil
	case uint:
		return float64(i), nil
	case string:
		return strconv.ParseFloat(i, 64)
	default:
		v := reflect.ValueOf(unk)
		v = reflect.Indirect(v)
		if v.Type().ConvertibleTo(floatType) {
			fv := v.Convert(floatType)
			return fv.Float(), nil
		} else if v.Type().ConvertibleTo(stringType) {
			sv := v.Convert(stringType)
			s := sv.String()
			return strconv.ParseFloat(s, 64)
		} else {
			return math.NaN(), fmt.Errorf("Can't convert %v to float64", v.Type())
		}
	}
}

func calculateCorrelation(s1, s2 dataframe.Series) float64 {
	var mean1, mean2 float64
	for i := 0; i < s1.NRows(); i++ {
		s1Value, err := getFloat(s1.Value(i))
		if err != nil {
			log.Fatal(err)
		}

		s2Value, err := getFloat(s2.Value(i))
		if err != nil {
			log.Fatal(err)
		}

		mean1 += s1Value
		mean2 += s2Value
	}
	mean1 /= float64(s1.NRows())
	mean2 /= float64(s2.NRows())

	var num, denom1, denom2 float64
	for i := 0; i < s1.NRows(); i++ {
		s1Value, err := getFloat(s1.Value(i))
		if err != nil {
			log.Fatal(err)
		}

		s2Value, err := getFloat(s2.Value(i))
		if err != nil {
			log.Fatal(err)
		}
		d1 := s1Value - mean1
		d2 := s2Value - mean2
		num += d1 * d2
		denom1 += d1 * d1
		denom2 += d2 * d2
	}

	return num / math.Sqrt(denom1*denom2)
}

func createCorrelationMatrix(df *dataframe.DataFrame) [][]float64 {
	corrMatrix := make([][]float64, len(df.Names()))
	for i := range corrMatrix {
		corrMatrix[i] = make([]float64, len(df.Names()))
	}

	// Calculate the correlation for each pair of columns
	for i := 0; i < len(df.Names()); i++ {
		for j := i; j < len(df.Names()); j++ {
			corr := calculateCorrelation(df.Series[i], df.Series[j])
			corrMatrix[i][j] = corr
			corrMatrix[j][i] = corr
		}
	}

	// Print the correlation matrix
	for _, row := range corrMatrix {
		for _, val := range row {
			fmt.Printf("%.2f ", val)
		}
		fmt.Println()
	}

	return corrMatrix
}

func saveCorrelationMatrixHeatmap(corrMatrix [][]float64, path string) {
	p, err := plot.New()
	if err != nil {
		log.Panic(err)
	}

	p.Title.Text = "Correlation Matrix Heatmap"

	heatmap := plotter.NewHeatMap(&GridXYZ{
		x: genSequence(0, 1, len(corrMatrix[0])),
		y: genSequence(0, 1, len(corrMatrix)),
		z: mat.NewDense(len(corrMatrix[0]), len(corrMatrix), flatten(corrMatrix)),
	}, palette.Heat(2, 1))

	// heatmap.Palette = moreland.Kindlmann() // Choose a color palette.
	p.Add(heatmap)

	if err := p.Save(10*vg.Inch, 10*vg.Inch, "heatmap.png"); err != nil {
		log.Panic(err)
	}
}

func genSequence(min, max, steps int) []float64 {
	seq := make([]float64, steps)
	step := float64(max-min) / float64(steps-1)
	for i := range seq {
		seq[i] = step * float64(i)
	}
	return seq
}

func flatten(mat [][]float64) []float64 {
	flattened := make([]float64, 0, len(mat)*len(mat[0]))
	for _, row := range mat {
		flattened = append(flattened, row...)
	}
	return flattened
}

func searchCorrelatedVariables(corrMatrix [][]float64, names []string, classIndex int) []string {
	var correlated []string
	for i := 0; i < len(corrMatrix); i++ {
		corr := corrMatrix[i][classIndex]
		if math.Abs(corr) > 0.3 {
			correlated = append(correlated, names[i])
		}
	}

	return correlated
}

// split splits DataFrame into two parts: train and test, according to the provided ratio
func split(df *dataframe.DataFrame, trainRatio float64, target string) (dataframe.DataFrame, dataframe.DataFrame, dataframe.DataFrame, dataframe.DataFrame) {
	xTrainSeries := []dataframe.Series{}
	yTrainSeries := []dataframe.Series{}
	xTestSeries := []dataframe.Series{}
	yTestSeries := []dataframe.Series{}

	for i, column := range df.Names() {
		series := df.Series[i]
		trainSerie := dataframe.NewSeriesInt64(column, nil)
		testSerie := dataframe.NewSeriesInt64(column, nil)

		for i := 0; i < series.NRows(); i++ {
			rowValue := series.Value(i).(int64)

			rowRatio := float64((float64(i) + 1.0) / float64(series.NRows()))

			if rowRatio < trainRatio {
				trainSerie.Append(rowValue)
			} else {
				testSerie.Append(rowValue)
			}

		}

		if column == target {

			yTrainSeries = append(yTrainSeries, trainSerie)
			yTestSeries = append(yTestSeries, testSerie)

			continue

		}

		xTrainSeries = append(xTrainSeries, trainSerie)
		xTestSeries = append(xTestSeries, testSerie)

	}

	xTrain := dataframe.NewDataFrame(xTrainSeries...)
	xTest := dataframe.NewDataFrame(xTestSeries...)
	yTrain := dataframe.NewDataFrame(yTrainSeries...)
	yTest := dataframe.NewDataFrame(yTestSeries...)

	return *xTrain, *xTest, *yTrain, *yTest
}

func newNeuralNetwork(inputDimension int, layer []int, activation deep.ActivationType, mode deep.Mode) *deep.Neural {
	return deep.NewNeural(&deep.Config{
		/* Input dimensionality */
		Inputs: inputDimension,
		/* Two hidden layers consisting of two neurons each, and a single output */
		Layout: layer, //[]int{2, 2, 1},
		/* Activation functions: Sigmoid, Tanh, ReLU, Linear */
		Activation: activation,
		/* Determines output layer activation & loss function:
		ModeRegression: linear outputs with MSE loss
		ModeMultiClass: softmax output with Cross Entropy loss
		ModeMultiLabel: sigmoid output with Cross Entropy loss
		ModeBinary: sigmoid output with binary CE loss */
		Mode: mode,
		/* Weight initializers: {deep.NewNormal(μ, σ), deep.NewUniform(μ, σ)} */
		Weight: deep.NewNormal(1.0, 0.0),
		/* Apply bias */
		Bias: true,
	})
}
