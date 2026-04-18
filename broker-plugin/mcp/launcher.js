const { spawn } = require('child_process');
const path = require('path');
const { platform, arch } = require('os');

const p = platform(), a = arch();
const name = p === 'win32'   ? 'skill_server_windows_amd64.exe'
           : p === 'darwin'  ? (a === 'arm64' ? 'skill_server_darwin_arm64' : 'skill_server_darwin_amd64')
           :                   (a === 'arm64' ? 'skill_server_linux_arm64'  : 'skill_server_linux_amd64');

const bin = path.join(__dirname, name);
const proc = spawn(bin, process.argv.slice(2), { stdio: 'inherit' });
proc.on('exit', code => process.exit(code ?? 0));
