# conchtalk-dlc

ConchTalk DLC (Downloadable Content) — a lightweight daemon that runs on your VPS to enable ConchTalk's relay mode.

## What it does

- Connects to ConchTalk's relay server via WebSocket
- Receives tool calls from the AI agent
- Executes commands locally on your server
- Streams results back in real-time

## Installation

```bash
bash <(curl -sL https://raw.githubusercontent.com/snana7mi/conchtalk-dlc/main/install.sh) -t <YOUR_TOKEN>
```

Generate your token in the ConchTalk iOS app under Server Settings → Relay Mode.

Custom server:

```bash
bash <(curl -sL https://raw.githubusercontent.com/snana7mi/conchtalk-dlc/main/install.sh) -t <TOKEN> -s wss://your-server.com/relay
```

## Manual Installation

```bash
curl -Lo conchtalk-dlc https://github.com/snana7mi/conchtalk-dlc/releases/latest/download/conchtalk-dlc-linux-amd64
chmod +x conchtalk-dlc
./conchtalk-dlc --token <YOUR_TOKEN>
```

## License

Apache-2.0
