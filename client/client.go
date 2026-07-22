package client

import (
	"tbox/client/config"
	"tbox/core"
	"tbox/core/manage"
	"tbox/core/node"
	"tbox/core/protocols"
	"tbox/core/setting"
	"tbox/log"
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hpcloud/tail"
	"github.com/olekukonko/tablewriter"
)

var (
	// coreCmd is the primary core process (the selected core in single-core mode;
	// sing-box in bridge mode).
	coreCmd *exec.Cmd
	// bridgeCmd is used only for TUN+xray bridge mode; it points at the xray
	// process acting as a protocol converter.
	bridgeCmd *exec.Cmd
	cmdMu     sync.Mutex
)

func Start(key string) {
	// Before starting, clean up stray cores not managed by tbox (e.g. sing-box
	// started directly by systemd, or a leftover from a previous run) so they
	// don't hog the socks port / tun interface and fail our startup.
	killStrayCores()
	testUrl := setting.TestUrl()
	testTimeout := setting.TestTimeout()
	manager := manage.Manager
	indexList := core.IndexList(key, manager.NodeLen())
	if len(indexList) == 0 {
		log.Warn("no node selected")
	} else if len(indexList) == 1 {
		index := indexList[0]
		node := manager.GetNode(index)
		manager.SetSelectedIndex(index)
		manager.Save()
		exe := run(node.Protocol)
		if exe {
			logStartSuccess(node.Protocol)
			result, status := TestNode(testUrl, setting.Socks(), testTimeout)
			log.Infof("%6s [ %s ] latency: %dms", status, testUrl, result)
		}
	} else {
		min := 100000
		i := -1
		for _, index := range indexList {
			node := manager.GetNode(index)
			exe := run(node.Protocol)
			if exe {
				result, status := TestNode(testUrl, setting.Socks(), testTimeout)
				log.Infof("%6s [ %s ] node: %d, latency: %dms", status, testUrl, index, result)
				if result > 0 && result <= setting.TestMinTime() {
					i = index
					min = result
					break
				}
				if result != -1 && min > result {
					i = index
					min = result
				}
			} else {
				return
			}
		}
		if i != -1 {
			log.Info("node with the lowest latency: ", i, ", latency: ", min, "ms")
			manager.SetSelectedIndex(i)
			manager.Save()
			node := manager.GetNode(i)
			exe := run(node.Protocol)
			if exe {
				logStartSuccess(node.Protocol)
			} else {
				log.Error("start failed")
			}
		} else {
			log.Info("none of the selected nodes can access the internet")
		}

	}
}

// Connecting tests the speed of each node in indexList one by one. Pass nil/empty
// to test all nodes. During testing it takes over SIGINT (Ctrl+C): a single
// press aborts the remaining tests and returns to the REPL without exiting the
// whole program; already-tested nodes keep their results.
func Connecting(indexList []int) {
	// Kill unmanaged stray cores first, so they don't fight for ports with the
	// short-lived cores we spin up for the speed test.
	killStrayCores()
	testUrl := setting.TestUrl()
	testTimeout := setting.TestTimeout()
	manager := manage.Manager
	if len(indexList) == 0 {
		indexList = core.IndexList("all", manager.NodeLen())
	}

	// Remember the currently selected node: the test reorders nodes, and we
	// restore the selection by content afterwards.
	var selectedLink string
	if n := manager.SelectedNode(); n != nil {
		selectedLink = n.GetLink()
	}

	// Temporarily take over Ctrl+C for the duration of the test: on signal,
	// just flip the interrupt flag and let the loop exit; signal.Stop restores
	// the default handler at the end so the shell is unaffected.
	interrupted := false
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		if _, ok := <-sigCh; ok {
			interrupted = true
		}
	}()
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
	}()

	tested := make([]*node.Node, 0, len(indexList))
	for _, index := range indexList {
		if interrupted {
			break
		}
		n := manager.GetNode(index)
		if n == nil {
			continue
		}
		exe := run(n.Protocol)
		if exe {
			result, _ := TestNode(testUrl, setting.Socks(), testTimeout)
			n.TestResult = float64(result)
		}
		Stop()
		tested = append(tested, n)
	}
	if interrupted {
		log.Warn("speed test aborted")
	}

	manager.NodeSort(func(n1 *node.Node, n2 *node.Node) bool {
		return n1.TestResult < n2.TestResult
	})
	manager.SetSelectedIndexByLink(selectedLink)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Index", "Protocol", "Alias", "Address", "Port", "Test Result"})
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	center := tablewriter.ALIGN_CENTER
	left := tablewriter.ALIGN_LEFT
	table.SetColumnAlignment([]int{center, center, left, center, center, center})
	table.SetColWidth(70)
	for _, index := range core.IndexList("all", manager.NodeLen()) {
		n := manager.GetNode(index)
		if n != nil {
			table.Append([]string{
				strconv.Itoa(index),
				string(n.GetProtocolMode()),
				n.GetName(),
				n.GetAddr(),
				strconv.Itoa(n.GetPort()),
				n.TestResultStr(),
			})
		}
	}
	if n := manager.SelectedNode(); n == nil {
		table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
			manager.SelectedIndex(),
			manager.NodeLen(),
			"no node",
		))
	} else {
		table.SetCaption(true, fmt.Sprintf("[ %d/%d ] %s",
			manager.SelectedIndex(),
			manager.NodeLen(),
			n.GetName(),
		))
	}
	table.Render()
}

func run(node protocols.Protocol) bool {
	Stop()
	switch node.GetProtocolMode() {
	case protocols.ModeShadowSocks, protocols.ModeTrojan, protocols.ModeVMess, protocols.ModeSocks, protocols.ModeVLESS, protocols.ModeVMessAEAD, protocols.ModeHysteria2, protocols.ModeShadowSocksR, protocols.ModeTUIC, protocols.ModeAnyTLS:
		// Pick the actual core to use based on the node's protocol/transport
		// (default sing-box; xhttp uses xray; Hysteria2/TUIC/AnyTLS/SSR use sing-box).
		coreName := config.SelectCore(node, setting.ClientCore())
		if coreName != CoreName {
			log.Infof("node %v automatically using core %v", node.GetProtocolMode(), coreName)
		}
		// Bridge decision: scan the main node and all split/DNS targets to find
		// the unique converter node. Multiple different xhttp/ssr nodes are
		// rejected; a unique converter node triggers dual-core bridge mode.
		plan, err := config.ResolveBridgePlan(node, setting.ClientCore())
		if err != nil {
			log.Error(err)
			return false
		}
		if plan.Enabled {
			return launchBridge(node, plan)
		}
		corePath, err := ResolveCore(coreName)
		if err != nil {
			log.Error(err)
			return false
		}
		if runtime.GOOS == "linux" && coreName == "sing-box" && setting.TunMode() {
			if !hasCapNetAdmin(corePath) {
				log.Warn("sing-box has not been granted CAP_NET_ADMIN, TUN mode may not work")
				log.Warn("please run: sudo setcap cap_net_admin,cap_net_bind_service=ep ", corePath)
			}
		}

		ok, ruleSetTimeout := launchCore(node, coreName, corePath)
		if ok {
			return true
		}
		// When sing-box rule-set downloads from GitHub time out (typical on
		// networks in mainland China; error looks like "initialize cache-file:
		// timeout"), automatically retry once with a mirror prefix. Once the
		// rule-set has been fetched successfully it is cached in cache.db, so
		// we clear the prefix after a successful retry — subsequent starts hit
		// the local cache and skip the mirror.
		if ruleSetTimeout && coreName == "sing-box" && config.RuleSetMirrorPrefix == "" {
			log.Warn("rule-set download timed out, retrying via mirror ", gitHubMirrorPrefix, " ...")
			config.RuleSetMirrorPrefix = gitHubMirrorPrefix
			ok2, _ := launchCore(node, coreName, corePath)
			config.RuleSetMirrorPrefix = ""
			if ok2 {
				log.Info("mirror-accelerated start succeeded; rule-set is cached, subsequent starts use the local cache")
				return true
			}
			return false
		}
		return false
	default:
		log.Infof("protocol %v is not supported yet", node.GetProtocolMode())
		return false
	}
}

// launchCore generates the config and starts the given core, blocking until the
// socks port is ready (success), the process exits (failure) or waitStartup
// elapses. Returns (success, suspected rule-set download timeout).
func launchCore(node protocols.Protocol, coreName, corePath string) (success bool, ruleSetTimeout bool) {
	cfg, err := config.CreateConfig(coreName)
	if err != nil {
		log.Error(err)
		return false, false
	}
	file, err := cfg.GenConfig(node)
	if err != nil {
		log.Error(err)
		return false, false
	}
	cmd, err := startCore(coreName, corePath, coreRunArgs(coreName, file))
	if err != nil {
		log.Error(err)
		return false, false
	}
	cmdMu.Lock()
	coreCmd = cmd
	cmdMu.Unlock()
	return waitCoreReady(cmd, coreName, setting.Socks(), setting.SetPid)
}

// coreRunArgs returns the args to run a core from a config file
// (works for both sing-box and xray): run -c <config>. For sing-box, when an
// extra config directory is set, appends -C.
func coreRunArgs(coreName, configPath string) []string {
	if coreName == "sing-box" && setting.SingboxConfigDir() != "" {
		return []string{"run", "-C", setting.SingboxConfigDir(), "-c", configPath}
	}
	return []string{"run", "-c", configPath}
}

// startCore builds and starts a core process with the given args: it stays
// resident detached from tbox's process group and its output is redirected to
// the unified log file. Returns the Start-ed *exec.Cmd. It does not touch any
// global state — the caller is responsible for storing it.
func startCore(coreName, corePath string, args []string) (*exec.Cmd, error) {
	cmd := exec.Command(corePath, args...)
	// xray needs geoip.dat / geosite.dat: explicitly set XRAY_LOCATION_ASSET to
	// point at the asset directory so we still find them when the binary lives
	// in a different directory (resolved from PATH/CORE_HOME).
	if coreName == "xray" {
		if assetDir := XrayAssetDir(corePath); assetDir != "" {
			cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+assetDir)
		}
	}
	// Detach the core from tbox's process group: when tbox exits (including
	// SIGINT delivered by the terminal to the foreground group on Ctrl+C) the
	// core is not affected and stays resident.
	detachProcess(cmd)
	// Write core output directly into the unified log file rather than piping
	// through tbox. That way, when tbox exits, the core won't be killed by a
	// SIGPIPE from writing into a closed pipe.
	logF, err := os.OpenFile(core.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open core log file: %w", err)
	}
	// The tbox parent process no longer needs this fd after Start (the core
	// already holds its own copy).
	defer logF.Close()
	fmt.Fprintf(logF, "==== [%s] run @ %s ====\n", coreName, time.Now().Format("2006-01-02 15:04:05"))
	cmd.Stdout = logF
	cmd.Stderr = logF
	if err = cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}

// waitCoreReady blocks until the core's port becomes ready (success), the
// process exits (failure), or waitStartup elapses. When ready or on slow-start
// it records the pid via setPid. Returns (success, suspected rule-set download timeout).
func waitCoreReady(cmd *exec.Cmd, coreName string, port int, setPid func(int) error) (success bool, ruleSetTimeout bool) {
	// Buffered: when we return early via the port-ready path, checkProc must
	// still be able to send without blocking once the process exits, so the
	// goroutine doesn't leak.
	status := make(chan struct{}, 1)
	go checkProc(cmd, status)

	// Poll for port readiness, up to waitStartup.
	// Port connectable = start succeeded; process exiting early = start failed.
	// waitStartup must exceed sing-box's own cache-file/rule-set download timeout
	// (~10s), otherwise we'd mistake a FATAL for a "slow background start success"
	// and miss both the error and the mirror-retry trigger.
	const waitStartup = 20 * time.Second
	deadline := time.NewTimer(waitStartup)
	defer deadline.Stop()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-status:
			// Process exited = start failed. Wait briefly so the core can flush
			// its final output to the log file.
			time.Sleep(100 * time.Millisecond)
			log.Error("failed to start ", coreName, " service, see the error output below to diagnose")
			runLines := readCurrentRunLog(coreName)
			for _, x := range runLines {
				log.Error(x)
			}
			return false, isRuleSetDownloadFailure(runLines)
		case <-deadline.C:
			// Timed out with no port yet, but the process is still alive:
			// possibly a slow start. Conservatively treat as success and record the pid.
			if cmd.Process != nil {
				setPid(cmd.Process.Pid)
			}
			log.Warnf("%s did not signal port ready within %v; continuing to start in the background", coreName, waitStartup)
			return true, false
		case <-ticker.C:
			if isPortReady(port) {
				if cmd.Process != nil {
					setPid(cmd.Process.Pid)
				}
				return true, false
			}
		}
	}
}

// launchBridge starts a "sing-box + converter" dual-core bridge: first the
// converter core (xray or ssr, providing a local socks inbound), then sing-box
// which handles the inbound/route/DNS/split. The converter hosts plan.Node
// (the unique converter node, which may be the main node or merely a split
// target); the sing-box frontend still uses mainNode as the "proxy" outbound
// (shape 2) or reuses the bridge (shape 1). If either core fails to start, we
// clean up whatever we already launched and return false.
func launchBridge(mainNode protocols.Protocol, plan *config.BridgePlan) bool {
	converter := plan.ConverterCore
	converterPath, err := ResolveCore(converter)
	if err != nil {
		log.Error(err)
		return false
	}
	singboxPath, err := ResolveCore("sing-box")
	if err != nil {
		log.Error(err)
		return false
	}
	if setting.TunMode() && !hasCapNetAdmin(singboxPath) {
		log.Warn("sing-box has not been granted CAP_NET_ADMIN, TUN mode may not work")
		log.Warn("please run: sudo setcap cap_net_admin,cap_net_bind_service=ep ", singboxPath)
	}

	bridgePort := config.BridgePort()

	// 1. Start the converter, hosting the converter node (plan.Node),
	// and wait for its internal socks port to be ready.
	converterArgs, err := bridgeConverterArgs(converter, plan.Node, bridgePort)
	if err != nil {
		log.Error(err)
		return false
	}
	ccmd, err := startCore(converter, converterPath, converterArgs)
	if err != nil {
		log.Error(err)
		return false
	}
	cmdMu.Lock()
	bridgeCmd = ccmd
	cmdMu.Unlock()
	if ok, _ := waitCoreReady(ccmd, converter, bridgePort, setting.SetBridgePid); !ok {
		Stop()
		return false
	}

	// 2. Start the sing-box frontend (its main outbound / split rules point at
	// bridgePort per plan), and wait for the external socks port to be ready.
	sbFile, err := config.SingBox{}.GenBridgeConfig(mainNode, plan, bridgePort)
	if err != nil {
		Stop()
		return false
	}
	scmd, err := startCore("sing-box", singboxPath, coreRunArgs("sing-box", sbFile))
	if err != nil {
		log.Error(err)
		Stop()
		return false
	}
	cmdMu.Lock()
	coreCmd = scmd
	cmdMu.Unlock()
	if ok, _ := waitCoreReady(scmd, "sing-box", setting.Socks(), setting.SetPid); !ok {
		Stop()
		return false
	}
	return true
}

// bridgeConverterArgs builds the launch args for the converter core.
//   - xray: writes a JSON config (single socks inbound + node outbound) and
//     starts with run -c
//   - ssr:  starts with CLI args -url <ssr link> -local 127.0.0.1:<bridgePort>;
//     under TUN mode we additionally bind the outbound so ssr's own traffic to
//     the node isn't hijacked back through the TUN loop.
func bridgeConverterArgs(converter string, node protocols.Protocol, bridgePort int) ([]string, error) {
	switch converter {
	case "xray":
		file, err := config.Xray{}.GenBridgeConfig(node, bridgePort)
		if err != nil {
			return nil, err
		}
		return coreRunArgs("xray", file), nil
	case "ssr":
		args := []string{
			"-url", node.GetLink(),
			"-local", fmt.Sprintf("127.0.0.1:%d", bridgePort),
		}
		return args, nil
	}
	return nil, fmt.Errorf("unknown bridge converter core: %s", converter)
}

// isRuleSetDownloadFailure reports whether the core output suggests a rule-set
// download failure/timeout. During cache-file initialization sing-box downloads
// remote rule-sets; direct GitHub access can time out with errors like
// "initialize cache-file: timeout". A match triggers a mirror-accelerated retry.
func isRuleSetDownloadFailure(lines []string) bool {
	for _, l := range lines {
		s := strings.ToLower(l)
		if !strings.Contains(s, "timeout") && !strings.Contains(s, "download") {
			continue
		}
		if strings.Contains(s, "cache-file") || strings.Contains(s, "cache_file") ||
			strings.Contains(s, "rule-set") || strings.Contains(s, "rule_set") ||
			strings.Contains(s, "ruleset") {
			return true
		}
	}
	return false
}

// logStartSuccess prints an accurate listening-port message based on the core
// actually in use. sing-box (1.12+) has only a mixed inbound: the HTTP proxy
// and SOCKS share one port and the separate http port setting has no effect;
// only xray has independent socks/http ports.
func logStartSuccess(node protocols.Protocol) {
	if plan, err := config.ResolveBridgePlan(node, setting.ClientCore()); err == nil && plan.Enabled {
		if plan.MainIsConverter {
			tunHint := ""
			if setting.TunMode() {
				tunHint = "TUN enabled, "
			}
			log.Infof("start succeeded, sing-box listening on socks/http port: %d (%smain outbound converted via %s), selected node: %d",
				setting.Socks(), tunHint, plan.ConverterCore, manage.Manager.SelectedIndex())
		} else {
			log.Infof("start succeeded, sing-box listening on socks/http port: %d (main node direct, split target node %d converted via %s), selected node: %d",
				setting.Socks(), plan.NodeIndex, plan.ConverterCore, manage.Manager.SelectedIndex())
		}
		if plan.ConverterCore == "xray" {
			log.Warn("note: xhttp transport does not support UDP, applications relying on UDP will not work")
		}
		return
	}
	coreName := config.SelectCore(node, setting.ClientCore())
	if coreName == "sing-box" {
		if setting.Http() > 0 {
			log.Warnf("sing-box's HTTP proxy shares port %d with SOCKS (mixed inbound); http port setting %d has no effect", setting.Socks(), setting.Http())
		}
		log.Infof("start succeeded, listening on socks/http port: %d, selected node: %d", setting.Socks(), manage.Manager.SelectedIndex())
		return
	}
	if setting.Http() == 0 {
		log.Infof("start succeeded, listening on socks port: %d, selected node: %d", setting.Socks(), manage.Manager.SelectedIndex())
	} else {
		log.Infof("start succeeded, listening on socks/http port: %d/%d, selected node: %d", setting.Socks(), setting.Http(), manage.Manager.SelectedIndex())
	}
}

func currentLogFile() string {
	// All core output is redirected to tbox.core.log, differentiated by
	// "[core-name] run" markers.
	return core.LogFile
}

// StopAll stops the cores managed by tbox and cleans up unmanaged stray cores
// (e.g. a sing-box launched directly by systemd). Called when the user runs the
// explicit stop command, ensuring the proxy is fully shut down.
func StopAll() {
	Stop()
	killStrayCores()
}

// Stop stops the service.
func Stop() {
	cmdMu.Lock()
	defer cmdMu.Unlock()
	if coreCmd != nil {
		coreCmd.Process.Kill()
		coreCmd = nil
	}
	if bridgeCmd != nil {
		bridgeCmd.Process.Kill()
		bridgeCmd = nil
	}
	if setting.Pid() != 0 {
		process, err := os.FindProcess(setting.Pid())
		if err == nil {
			process.Kill()
		}
		setting.SetPid(0)
	}
	// Also clean up the pid of the xray converter used in bridge mode
	// (common after a tbox restart when coreCmd/bridgeCmd have been lost
	// and we can only reap by recorded pid).
	if setting.BridgePid() != 0 {
		process, err := os.FindProcess(setting.BridgePid())
		if err == nil {
			process.Kill()
		}
		setting.SetBridgePid(0)
	}
	// Remove the log file if it has grown too large.
	logFile := currentLogFile()
	file, err := os.Stat(logFile)
	if err == nil && file != nil {
		fileSize := float64(file.Size()) / (1 << 20)
		if fileSize > 5 {
			os.Remove(logFile)
		}
	}
}

// killStrayCores kills stray core processes not managed by the current tbox
// process. Scenarios: sing-box started directly by systemd, or a leftover core
// from an abnormal previous run — their PIDs are not stored in settings and
// Stop's normal logic cannot see them. Without cleanup, a new core would fail
// to start because the socks port or the tun interface is already occupied.
// Matches by the resolved absolute core binary path (pkill -f full-command-line)
// to avoid killing unrelated same-named programs. Only runs on Unix-like systems
// (depends on pkill).
func killStrayCores() {
	if runtime.GOOS == "windows" {
		return
	}
	seen := make(map[string]struct{})
	for _, corePath := range coreCache {
		corePath = strings.TrimSpace(corePath)
		if corePath == "" {
			continue
		}
		if _, ok := seen[corePath]; ok {
			continue
		}
		seen[corePath] = struct{}{}
		// -f matches the full command line; the pattern is the absolute binary
		// path, ensuring we only hit "<corePath> run ..." processes. tbox itself
		// is not resident under that path in run mode, so there's no self-kill risk.
		cmd := exec.Command("pkill", "-f", corePath+" run")
		// pkill exits with code 1 when nothing matches, which is normal — ignore the error.
		_ = cmd.Run()
	}
}

// View the core log.
// First prints the tail of the existing log (so a just-happened startup FATAL
// is visible), then tails new content in real time; press Enter to exit
// (no need for Ctrl+C).
func ShowLog() {
	logFile := currentLogFile()
	printLogTail(logFile, 50)
	fmt.Println("---- tailing log in real time, press Enter to exit ----")

	t, err := tail.TailFile(logFile, tail.Config{
		ReOpen:    true,                                 // reopen file when rotated
		Follow:    true,                                 // follow new content
		Location:  &tail.SeekInfo{Offset: 0, Whence: 2}, // start from the end of the file
		MustExist: false,                                // don't error if the file doesn't exist
		Poll:      true,
		Logger:    tail.DiscardingLogger, // suppress the tail library's own log output
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer t.Stop()

	go func() {
		for line := range t.Lines {
			fmt.Println(line.Text)
		}
	}()

	// Block the main goroutine waiting for Enter to exit; ishell is not
	// reading stdin at this point, so we can read directly.
	bufio.NewReader(os.Stdin).ReadString('\n')
}

// printLogTail prints up to n lines from the tail of the log file. The log
// file is capped at 5MB, so reading it whole is not a concern.
func printLogTail(logFile string, n int) {
	data, err := os.ReadFile(logFile)
	if err != nil {
		return
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}
	for _, line := range lines[start:] {
		if line != "" {
			fmt.Println(line)
		}
	}
}

// readCurrentRunLog reads the output lines written by the current core startup
// to the log file (up to 20 lines). Core output goes to core.LogFile, and
// before every startup we write an "==== [core] run @ ... ====" marker. This
// reads from just after the last marker, used for diagnosing startup failures.
func readCurrentRunLog(coreName string) []string {
	data, err := os.ReadFile(core.LogFile)
	if err != nil {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	marker := fmt.Sprintf("==== [%s] run @", coreName)
	start := 0
	for i, l := range lines {
		if strings.HasPrefix(l, marker) {
			start = i + 1
		}
	}
	result := make([]string, 0, 20)
	for _, l := range lines[start:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		result = append(result, l)
		if len(result) >= 20 {
			break
		}
	}
	return result
}

func hasCapNetAdmin(path string) bool {
	if runtime.GOOS != "linux" {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	mode := info.Mode()
	return mode&0100 != 0 || os.Getuid() == 0
}

// check process status
func checkProc(c *exec.Cmd, status chan struct{}) {
	c.Wait()
	status <- struct{}{}
}

// isPortReady probes whether the local socks port can be connected (core has bound the listener).
func isPortReady(port int) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), 200*time.Millisecond)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
