// Package try provides retry functionality.
//     var value string
//     err := try.Do(func(attempt int) (retry bool, err error) {
//       var err error
//       value, err = SomeFunction()
//       return attempt < 5, err // try 5 times
//     })
//     if err != nil {
//       log.Error(err, "somefunction failed")
//     }
//
// This package was created by Mat Ryer and modified to support multierr, the
// original package is available at:
//   https://github.com/matryer/try
// With the post introducing it at:
//   https://medium.com/@matryer/retrying-in-golang-quicktip-f688d00e650a#.3r2nbnjwu
package try
