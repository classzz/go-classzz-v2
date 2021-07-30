// Copyright 2015 The go-classzz-v2 Authors
// This file is part of the go-classzz-v2 library.
//
// The go-classzz-v2 library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-classzz-v2 library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-classzz-v2 library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/classzz/go-classzz-v2/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("czz/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("czz/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("czz/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("czz/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("czz/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("czz/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("czz/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("czz/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("czz/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("czz/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("czz/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("czz/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("czz/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("czz/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("czz/downloader/throttle", nil)
)
