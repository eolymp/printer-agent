# Printer Agent

The Printer Agent is a utility that acts as a bridge between your printer and the Eolymp server, enabling you to connect a printer to the Eolymp platform. This allows participants to print their solutions directly from the contest page.

## Installation

You can download the latest version of the agent from the [releases page](https://github.com/eolymp/printer-agent/releases). Make sure to select the correct version for your operating system and architecture.

Alternatively, you can install the Printer Agent directly from the source using the `go install` command:

```bash
go install github.com/eolymp/printer-agent@latest
```

This will install the latest version of the agent.

## Usage

Before running the agent, you need to configure your printer in the [Eolymp Console](https://console.eolymp.com/). Currently, this feature is available only by request, so if you wish to use it, please [contact us](mailto:support@eolymp.com).

To run the agent, you'll need to gather a few parameters:

- **Space ID**: A unique identifier for your space on Eolymp. You can find this on the Settings page of your space.
- **Printer Token**: A unique token used to authenticate the printer connection. You can find this on the Printers page of your space. To see the token, add or edit a printer and you’ll find it under the token field (it begins with `eprt-`).
- **Printer URI**: The URI of the printer you wish to connect to, using the IPP protocol. Run the agent in discovery mode to find available printers in your network.

Once you have the required parameters, run the following command:

```bash
printer-agent \
  -space=0se12sf \
  -token=eprn-11examplexx0 \
  -printer=ipp://localhost:631/printers/My_Printer
```

The agent will connect to both the printer and the Eolymp server, forwarding printer status updates and accepting print jobs from the server.

## How-To Guides

### How to Get Your Space ID

The Space ID is a unique identifier for your space on Eolymp. You can find this ID on the Settings page of your space. Be mindful that the Space ID is different from the Space Key (identifier used in the address of space's site).

![A screenshot showing Space ID on Settings page](.github/readme/space-id.png)

### How to Get Your Printer Token

The Printer Token is used for authenticating the printer connection to Eolymp. You can find this token on the Printers page of your space. To see the token, add a new printer or edit an existing one. The token will begin with `eprt-`.

![A screenshot showing Printer Token in a dialog](.github/readme/printer-token.png)

### How to Get Your Printer URI

The Printer URI specifies the location of the printer, following the IPP protocol. You can discover available printers in your network by running the Printer Agent in discovery mode (without any flags):

```bash
$ printer-agent
You should provide -printer option to connect to the printer and start printing documents.
Learn more: https://github.com/eolymp/printer-agent/blob/main/README.md

Looking for available printers...
- ipp://localhost:631/printers/My_Printer (idle)
```

If your printer supports the IPP protocol natively, you can use the provided URI. If not, you may need to install an IPP server like [CUPS](https://www.cups.org/).

For printers that require authentication, you can specify the username and password in the URI, such as:

```bash
ipp://username:password@localhost:631/printers/My_Printer
```

or if SSL/TLS is required:

```bash
ipps://username:password@localhost:631/printers/My_Printer
```

### How to Enable IPP Server on macOS

If you're using macOS, you can enable the built-in IPP server by following these steps:
1. Go to **System Preferences → Printers & Scanners**.
2. Choose your printer.
3. Enable **Share this printer on the network**.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contact

For support or inquiries, please reach out to us at [support@eolymp.com](mailto:support@eolymp.com).
