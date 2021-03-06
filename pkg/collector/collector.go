/*
Copyright 2020 Christian Niehoff.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"bytes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/r3kzi/clamav-prometheus-exporter/pkg/clamav"
	"github.com/r3kzi/clamav-prometheus-exporter/pkg/commands"
	"regexp"
	"strconv"
	log "github.com/sirupsen/logrus"
)

//Collector satisfies prometheus.Collector interface
type Collector struct {
	client      clamav.Client
	up          *prometheus.Desc
	threadsLive *prometheus.Desc
	threadsIdle *prometheus.Desc
	threadsMax  *prometheus.Desc
	queue       *prometheus.Desc
	memHeap     *prometheus.Desc
	memMmap     *prometheus.Desc
	memUsed     *prometheus.Desc
	buildInfo   *prometheus.Desc
	poolsUsed   *prometheus.Desc
	poolsTotal  *prometheus.Desc
}

//New creates a Collector struct
func New(client clamav.Client) *Collector {
	return &Collector{
		client:      client,
		up:          prometheus.NewDesc("clamav_up", "Shows UP Status", nil, nil),
		threadsLive: prometheus.NewDesc("clamav_threads_live", "Shows live threads", nil, nil),
		threadsIdle: prometheus.NewDesc("clamav_threads_idle", "Shows idle threads", nil, nil),
		threadsMax:  prometheus.NewDesc("clamav_threads_max", "Shows max threads", nil, nil),
		queue:       prometheus.NewDesc("clamav_queue", "Shows queued items", nil, nil),
		memHeap:     prometheus.NewDesc("clamav_mem_heap", "Shows heap memory usage", nil, nil),
		memMmap:     prometheus.NewDesc("clamav_mem_mmap", "Shows mmap memory usage", nil, nil),
		memUsed:     prometheus.NewDesc("clamav_mem_used", "Shows used memory usage", nil, nil),
		buildInfo:   prometheus.NewDesc("clamav_build_info", "Shows ClamAV Build Info", []string{"clamav_version", "database_version"}, nil),
		poolsUsed:   prometheus.NewDesc("clamav_pools_used_mb", "Shows pools usage memory in Mb", nil, nil),
		poolsTotal:  prometheus.NewDesc("clamav_pools_total_mb", "Shows pools total memory in Mb", nil, nil),
	}
}

//Describe satisfies prometheus.Collector.Describe
func (collector *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.up
	ch <- collector.threadsLive
	ch <- collector.threadsIdle
	ch <- collector.threadsMax
	ch <- collector.queue
	ch <- collector.memHeap
	ch <- collector.memMmap
	ch <- collector.memUsed
	ch <- collector.buildInfo
	ch <- collector.poolsUsed
	ch <- collector.poolsTotal
}

//Collect satisfies prometheus.Collector.Collect
func (collector *Collector) Collect(ch chan<- prometheus.Metric) {
	pong := collector.client.Dial(commands.PING)
	if bytes.Compare(pong, []byte{'P', 'O', 'N', 'G', '\n'}) == 0 {
		ch <- prometheus.MustNewConstMetric(collector.up, prometheus.CounterValue, 1)
	}

	float := func(s string) float64 {
		float, err := strconv.ParseFloat(s, 64)
		if err != nil {
			log.Fatalf("couldn't parse string to float: %s", err)
		}
		return float
	}
// example of output	
/*
echo -n  -e 'zSTATS\0' | nc 127.0.0.1 3310
POOLS: 1

STATE: VALID PRIMARY
THREADS: live 10  idle 2 max 10 idle-timeout 30
QUEUE: 0 items
	INSTREAM 1.249366 instream(10.42.174.58@41938)
	IDLE -0.021568 
	IDLE 0.022133 
	INSTREAM 11.672077 instream(10.42.174.58@35486)
	INSTREAM 6.629173 instream(10.42.174.58@38126)
	INSTREAM 6.627400 instream(10.42.174.58@38128)
	INSTREAM 1.606307 instream(10.42.174.58@41684)
	STATS 0.000276 
	IDLE 0.013437 
	INSTREAM 0.010599 instream(10.42.74.224@39776)

MEMSTATS: heap N/A mmap N/A used N/A free N/A releasable N/A pools 1 pools_used 1143.596M pools_total 1143.632M
END
*/
	stats := collector.client.Dial(commands.STATS)
	log.Debug("Stats: " + string(stats))

	// THREADS:
	regex := regexp.MustCompile("THREADS: live ([0-9]+)  idle ([0-9]+) max ([0-9]+) ")
	matches := regex.FindAllStringSubmatch(string(stats), -1)
	if len(matches) > 0 {
		ch <- prometheus.MustNewConstMetric(collector.threadsLive, prometheus.GaugeValue, float(matches[0][1]))
		ch <- prometheus.MustNewConstMetric(collector.threadsIdle, prometheus.GaugeValue, float(matches[0][2]))
		ch <- prometheus.MustNewConstMetric(collector.threadsMax, prometheus.GaugeValue, float(matches[0][3]))
	}

	// QUEUE:
	regex = regexp.MustCompile("QUEUE:: ([0-9]+) items")
	matches  = regex.FindAllStringSubmatch(string(stats), -1)
	if len(matches) > 0 {
		ch <- prometheus.MustNewConstMetric(collector.queue, prometheus.GaugeValue, float(matches[0][1]))
	}

	// MEMSTATS:
	regex = regexp.MustCompile("MEMSTATS: heap (N/A|[0-9]+) mmap (N/A|[0-9]+) used (N/A|[0-9]+) free (N/A|[0-9]+) releasable (N/A|[0-9]+) pools ([0-9]+) pools_used ([0-9.]+)M pools_total ([0-9.]+)M")
	matches  = regex.FindAllStringSubmatch(string(stats), -1)
	if len(matches) > 0 {
		// for i := 0; i < len(matches[0]); i++ {
		// 	log.Info("MEMSTATS: " + string(matches[0][i]))
		// }
		if matches[0][1] != "N/A"{
			ch <- prometheus.MustNewConstMetric(collector.memHeap, prometheus.GaugeValue, float(matches[0][1]))
		}
		if matches[0][2] != "N/A"{
			ch <- prometheus.MustNewConstMetric(collector.memMmap, prometheus.GaugeValue, float(matches[0][2]))
		}
		if matches[0][3] != "N/A"{
			ch <- prometheus.MustNewConstMetric(collector.memUsed, prometheus.GaugeValue, float(matches[0][3]))
		}
		ch <- prometheus.MustNewConstMetric(collector.poolsUsed, prometheus.GaugeValue, float(matches[0][7]))
		ch <- prometheus.MustNewConstMetric(collector.poolsTotal, prometheus.GaugeValue, float(matches[0][8]))
	}

	
	

	version := collector.client.Dial(commands.VERSION)
	regex = regexp.MustCompile("((ClamAV)+\\s([0-9.]*)/([0-9.]*))")
	matches = regex.FindAllStringSubmatch(string(version), -1)
	if len(matches) > 0 {
		ch <- prometheus.MustNewConstMetric(collector.buildInfo, prometheus.GaugeValue, 1, matches[0][3], matches[0][4])
	}

}
