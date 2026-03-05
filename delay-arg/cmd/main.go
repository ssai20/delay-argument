package main

import (
	"delay-argument-go/internal/differenceScheme"
	"delay-argument-go/internal/examineSolution"
	"delay-argument-go/internal/gridDesign"
	"delay-argument-go/internal/latex"
	"delay-argument-go/internal/thomasMethod"
	"fmt"
	"math"
)

func solution(x float64, e float64) float64 {
	return math.Cos(math.Pi*x/2.) + math.Exp(-x/e)
}
func function(x, e, d float64) float64 {
	return -math.Cos(math.Pi*x/2.)*(math.Pi*math.Pi*e/4.) - math.Pi/2.*math.Sin(math.Pi*x/2.) -
		-math.Exp((d-x)/e) - math.Cos(math.Pi*(x-d)/2.)
}
func phi(x float64, e float64) float64 {
	return math.Exp(-x / e)
}

func main() {
	classic := make([][]string, 9)
	modified := make([][]string, 9)

	i := 0
	for e := 1.; e >= 1.e-08; e /= 10. {
		classic[i] = make([]string, 5)
		modified[i] = make([]string, 5)
		j := 0
		for n := 128; n <= 2048; n *= 2 {
			d := 0.
			fn := func(x float64) float64 {
				return function(x, e, d)
			}
			sl := func(x float64) float64 {
				return solution(x, e)
			}
			h := gridDesign.ShishkinMesh(e, n)
			uzel := gridDesign.FindPoints(h, n)
			abcf1 := differenceScheme.ClassicTeylorFormulasScheme(n, e, h, d, fn, uzel)
			abcf2 := differenceScheme.ClassicTeylorFormulasScheme(n, e, h, d, fn, uzel)
			u1 := thomasMethod.Progonka(abcf1.A, abcf1.B, abcf1.C, abcf1.F, n, e)
			u2 := thomasMethod.Progonka(abcf2.A, abcf2.B, abcf2.C, abcf2.F, n, e)
			a := examineSolution.ErrorNorm(u1, n, sl, uzel)
			b := examineSolution.ErrorNorm(u2, n, sl, uzel)

			classic[i][j] = fmt.Sprintf("%6.2e", a)
			modified[i][j] = fmt.Sprintf("%6.2e", b)
			j++
			fmt.Printf("epsilon=%e, n=%d\n", e, n)
			fmt.Printf("h[0]=%e, h[%d]=%e\n", h[0], n, h[n])
		}
		i++
	}

	latex.Latex(
		"/home/funforces/Articles/Trailing/доклад/результаты/06032026/rn-delta-00eps1-001eps-1-08-N-128-2048-116-zadorin.tex",
		"сетка Шишкина $\\delta = 0.0$",
		classic,
		modified)
}
