#!/usr/bin/env python3
"""Simple test Python application."""

def main():
    print("🐍 Python app running!")
    print("Press Ctrl+C to stop.")
    
    # Keep running
    import time
    try:
        while True:
            time.sleep(1)
    except KeyboardInterrupt:
        print("\n👋 Bye!")

if __name__ == "__main__":
    main()
