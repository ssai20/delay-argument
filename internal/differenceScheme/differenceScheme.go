package differenceScheme

//type DifferenceCoefficients struct {
//	A []float64
//	B []float64
//	C []float64
//	F []float64
//	//N int
//}

//func NewDifferenceScheme(N int) *DifferenceCoefficients {
//	return &DifferenceCoefficients{
//		A: make([]float64, N),
//		B: make([]float64, N),
//		C: make([]float64, N),
//		F: make([]float64, N),
//		N: N,
//	}
//}

type ABCF struct {
	A, B, C, F []float64
}

func ClassicTeylorFormulasScheme(
	N int,
	epsilon float64,
	h []float64,
	delta float64,
	fn func(float64) float64,
	uzel []float64) ABCF {
	T := epsilon - delta*delta/2.

	A := make([]float64, N+1)
	B := make([]float64, N+1)
	C := make([]float64, N+1)
	F := make([]float64, N+1)

	for i := 1; i < N; i++ {
		A[i] = 2. * T / (h[i] + h[i+1]) / h[i]
		B[i] = -(2.*T/h[i]/h[i+1] + (delta+1.)/h[i+1] + 1.)
		C[i] = 2.*T/(h[i]+h[i+1])/h[i+1] + (1.+delta)/h[i+1]
		F[i] = fn(uzel[i])
	}

	return ABCF{
		A,
		B,
		C,
		F,
	}
}
