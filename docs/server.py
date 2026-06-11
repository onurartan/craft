#!/usr/bin/env python3
"""
Craft Documentation Local Server
================================

WHY IS THIS NEEDED?
When developing or testing the documentation locally, you cannot simply double-click 
'docs.html' to open it in a browser (using the file:// protocol). Browsers block local 
AJAX/Fetch requests for security reasons (CORS policy), meaning the sidebar will fail 
to load the Markdown files from the '../.docs' directory.

This script spins up a lightweight, zero-dependency Python HTTP server that correctly 
serves the files over http://localhost, allowing all local Fetch requests to work 
perfectly without needing to install heavy tools like VSCode LiveServer or Node.js.

USAGE:
  python docs/server.py
  python docs/server.py --port 8080

  # start server and view only .md files (if you are in the project root)
  python docs/server.py --md-view CRAFT.md

  OR

  uv run docs/server.py
  uv run docs/server.py --port 8080

  # start server and view only .md files (if you are in the project root)
  uv run docs/server.py --md-view CRAFT.md
"""

import os
import sys
import time
import argparse
import webbrowser
from http.server import SimpleHTTPRequestHandler
# Use ThreadingHTTPServer so long-polling doesn't block other requests
try:
    from http.server import ThreadingHTTPServer as HTTPServer
except ImportError:
    from http.server import HTTPServer

# --- ANSI Colors for Pro UI ---
class Colors:
    CYAN = '\033[96m'
    GREEN = '\033[92m'
    YELLOW = '\033[93m'
    MAGENTA = '\033[95m'
    RESET = '\033[0m'
    BOLD = '\033[1m'

def print_banner(port, path, md_view=None):
    os.system('cls' if os.name == 'nt' else 'clear')
    
    print()
    print(f"  {Colors.CYAN}{Colors.BOLD}▲ Craft Docs Server{Colors.RESET} {Colors.MAGENTA}v1.0{Colors.RESET}")
    print()
    if md_view:
        print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}Mode:{Colors.RESET}     {Colors.MAGENTA}Standalone Preview{Colors.RESET}")
        print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}Local:{Colors.RESET}    {Colors.CYAN}http://localhost:{port}/preview{Colors.RESET}")
        print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}File:{Colors.RESET}     {md_view}")
    else:
        print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}Local:{Colors.RESET}    {Colors.CYAN}http://localhost:{port}/docs/docs.html{Colors.RESET}")
        print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}Root:{Colors.RESET}     {path}")
    
    print(f"  {Colors.GREEN}➜{Colors.RESET}  {Colors.BOLD}Reload:{Colors.RESET}   {Colors.YELLOW}Enabled{Colors.RESET}")
    print()
    print(f"  {Colors.RESET}Press {Colors.BOLD}Ctrl+C{Colors.RESET} to stop the server")
    print()

# --- Live Reload Logic ---
def get_latest_mtime(watch_dirs, watch_file=None, extensions=('.md', '.html', '.css', '.js')):
    latest = 0
    # Check specific file first
    if watch_file:
        try:
            mtime = os.path.getmtime(watch_file)
            if mtime > latest:
                latest = mtime
        except OSError:
            pass
            
    # Check directories
    for d in watch_dirs:
        for root, _, files in os.walk(d):
            for f in files:
                if f.endswith(extensions):
                    path = os.path.join(root, f)
                    try:
                        mtime = os.path.getmtime(path)
                        if mtime > latest:
                            latest = mtime
                    except OSError:
                        pass
    return latest

LIVERELOAD_JS = b"""
<!-- INJECTED BY CRAFT SERVER -->
<script>
(function() {
    let lastMtime = null;
    function poll() {
        fetch('/_livereload')
            .then(res => res.text())
            .then(mtime => {
                if (lastMtime && lastMtime !== mtime) {
                    console.log(' File changed, reloading...');
                    window.location.reload();
                }
                lastMtime = mtime;
                setTimeout(poll, 1000); // Check every second
            })
            .catch(() => setTimeout(poll, 2000));
    }
    poll();
})();
</script>
"""

class CraftDocHandler(SimpleHTTPRequestHandler):
    # Watch both the root (for index) and .docs (for markdown files)
    WATCH_DIRS = []
    WATCH_FILE = None
    PROJECT_ROOT = ""

    def do_GET(self):
        # Handle the custom live-reload long-polling endpoint
        if self.path == '/_livereload':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.send_header('Cache-Control', 'no-cache')
            self.end_headers()
            latest = get_latest_mtime(self.WATCH_DIRS, self.WATCH_FILE)
            self.wfile.write(str(latest).encode())
            return
            
        # Handle standalone markdown preview route
        if self.path == '/preview' and self.WATCH_FILE:
            try:
                # Read mdpreview.html
                preview_path = os.path.join(self.PROJECT_ROOT, 'docs', 'mdpreview.html')
                with open(preview_path, 'r', encoding='utf-8') as f:
                    html_content = f.read()
                
                # Read the target markdown file
                try:
                    with open(self.WATCH_FILE, 'r', encoding='utf-8') as f:
                        md_content = f.read()
                except OSError as e:
                    md_content = f"# Error Loading File\n\nCould not read: `{self.WATCH_FILE}`\n\nError: {str(e)}"
                
                # Escape backticks and backslashes to prevent breaking the JS string if we were using variables
                # But since we use <script type="text/markdown"> we just replace the placeholder directly.
                # Safe injection:
                html_content = html_content.replace('{{MARKDOWN_CONTENT}}', md_content)
                
                # Inject Livereload JS
                html_content = html_content.replace('</body>', LIVERELOAD_JS.decode('utf-8') + '\n</body>')
                
                content_bytes = html_content.encode('utf-8')
                self.send_response(200)
                self.send_header('Content-type', 'text/html')
                self.send_header('Content-Length', str(len(content_bytes)))
                self.end_headers()
                self.wfile.write(content_bytes)
                return
            except Exception as e:
                self.send_response(500)
                self.end_headers()
                self.wfile.write(f"Internal Server Error: {e}".encode())
                return

        # Serve static files, but inject JS into HTML
        path = self.translate_path(self.path)
        if os.path.isfile(path) and path.endswith('.html'):
            try:
                with open(path, 'rb') as f:
                    content = f.read()
                # Inject the script right before closing body or at the end
                if b'</body>' in content:
                    content = content.replace(b'</body>', LIVERELOAD_JS + b'</body>')
                else:
                    content += LIVERELOAD_JS
                
                self.send_response(200)
                self.send_header('Content-type', 'text/html')
                self.send_header('Content-Length', str(len(content)))
                self.end_headers()
                self.wfile.write(content)
                return
            except OSError:
                pass # Fall back to default handler if reading fails
                
        # For all other files, use default serving
        super().do_GET()

    def log_message(self, format, *args):
        try:
            # Clean CLI formatting: Only log 404s or actual file accesses, ignore /_livereload spam
            if isinstance(args[0], str) and args[0].startswith("GET /_livereload"):
                return
            
            msg = format % args
            status = str(args[1]) if len(args) > 1 else str(args[0])
            
            if status.startswith("2") or status.startswith("3"):
                color = Colors.GREEN
            elif status.startswith("4") or status.startswith("5"):
                color = Colors.YELLOW
            else:
                color = Colors.RESET
                
            sys.stderr.write(f"{color}[CRAFT DOCS] {msg}{Colors.RESET}\n")
        except Exception:
            # Fallback for unexpected log formats
            sys.stderr.write(f"[CRAFT DOCS] {format % args}\n")

def main():
    # Force UTF-8 encoding for Windows terminals to support symbols like ▲ and ➜
    if sys.stdout.encoding.lower() != 'utf-8':
        sys.stdout.reconfigure(encoding='utf-8')

    parser = argparse.ArgumentParser(description="Start a local server for Craft documentation.")
    parser.add_argument("--port", "-p", type=int, default=8000)
    parser.add_argument("--md-view", type=str, help="Preview a specific markdown file in standalone mode.")
    args = parser.parse_args()

    script_dir = os.path.dirname(os.path.abspath(__file__))
    project_root = os.path.dirname(script_dir)
    
    # IMPORTANT: Resolve the md_view path BEFORE changing directory to project_root, 
    # otherwise relative paths like '../CRAFT.md' will be calculated incorrectly.
    target_file = None
    if args.md_view:
        # First try: resolve relative to current working directory (where user ran the command)
        target_file = os.path.abspath(args.md_view)
        # Fallback: If it doesn't exist, maybe they wrote the path relative to the script's directory (docs/)
        if not os.path.exists(target_file):
            fallback_path = os.path.abspath(os.path.join(script_dir, args.md_view))
            if os.path.exists(fallback_path):
                target_file = fallback_path
        
    os.chdir(project_root)
    CraftDocHandler.PROJECT_ROOT = project_root

    if target_file:
        # Standalone mode: Watch only the mdpreview.html and the target file
        CraftDocHandler.WATCH_FILE = target_file
        CraftDocHandler.WATCH_DIRS = [os.path.join(project_root, 'docs')]
    else:
        # Full docs mode: Watch both the docs html folder and the .docs markdown folder
        CraftDocHandler.WATCH_DIRS = [
            os.path.join(project_root, 'docs'),
            os.path.join(project_root, '.docs')
        ]

    server_address = ('', args.port)
    try:
        httpd = HTTPServer(server_address, CraftDocHandler)
    except OSError as e:
        print(f"\n{Colors.YELLOW}[ERROR] Failed to bind to port {args.port}: {e}{Colors.RESET}")
        sys.exit(1)

    print_banner(args.port, project_root, md_view=CraftDocHandler.WATCH_FILE)

    if args.md_view:
        url = f"http://localhost:{args.port}/preview"
    else:
        url = f"http://localhost:{args.port}/docs/docs.html"
        
    try:
        webbrowser.open(url)
    except Exception:
        pass

    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print(f"\n{Colors.MAGENTA}🛑 Server gracefully stopped.{Colors.RESET}")
        httpd.server_close()
        sys.exit(0)

if __name__ == '__main__':
    main()