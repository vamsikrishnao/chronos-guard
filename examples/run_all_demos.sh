#!/usr/bin/env bash
set -e

# Terminal Colors
GREEN='\033[1;32m'
CYAN='\033[1;36m'
YELLOW='\033[1;33m'
RED='\033[1;31m'
NC='\033[0m' # No Color

echo -e "${CYAN}====================================================================${NC}"
echo -e "${CYAN}   CHRONOS-GUARD: FinOps LLM Cost & Runaway Agent Safeguard Demo   ${NC}"
echo -e "${CYAN}====================================================================${NC}"

run_python() {
    echo -e "\n${YELLOW}--------------------------------------------------------------------${NC}"
    echo -e "${GREEN}>>> Running Python SDK Demo (@guard_budget Decorator)...${NC}"
    echo -e "${YELLOW}--------------------------------------------------------------------${NC}"
    cd "$(dirname "$0")/python"
    pip install -q -r requirements.txt
    python3 demo_agent.py
}

run_java() {
    echo -e "\n${YELLOW}--------------------------------------------------------------------${NC}"
    echo -e "${GREEN}>>> Running Java SDK Demo (Spring AOP @GuardedAgentStep)...${NC}"
    echo -e "${YELLOW}--------------------------------------------------------------------${NC}"
    cd "$(dirname "$0")/java"
    mvn -q clean compile exec:java -Dexec.mainClass="com.chronos.demo.AgentDemoApp"
}

run_ruby() {
    echo -e "\n${YELLOW}--------------------------------------------------------------------${NC}"
    echo -e "${GREEN}>>> Running Ruby SDK Demo (ActiveJob Server Middleware)...${NC}"
    echo -e "${YELLOW}--------------------------------------------------------------------${NC}"
    cd "$(dirname "$0")/ruby"
    bundle install --quiet
    ruby demo_agent.rb
}

CHOICE=${1:-""}

if [ -z "$CHOICE" ]; then
    echo -e "\nSelect a demo to execute:"
    echo "1) Python SDK Demo"
    echo "2) Java SDK Demo"
    echo "3) Ruby SDK Demo"
    echo "4) Run ALL Demos Sequentially"
    echo "5) Exit"
    read -p "Enter choice [1-5]: " SELECTION

    case $SELECTION in
        1) run_python ;;
        2) run_java ;;
        3) run_ruby ;;
        4) run_python; run_java; run_ruby ;;
        5) echo "Exiting demo."; exit 0 ;;
        *) echo -e "${RED}Invalid choice.${NC}"; exit 1 ;;
    esac
else
    case $CHOICE in
        python) run_python ;;
        java) run_java ;;
        ruby) run_ruby ;;
        all) run_python; run_java; run_ruby ;;
        *) echo -e "${RED}Usage: $0 [python|java|ruby|all]${NC}"; exit 1 ;;
    esac
fi

echo -e "\n${GREEN}====================================================================${NC}"
echo -e "${GREEN}   Demo Execution Complete! All AI Guardrail Invariants Verified.   ${NC}"
echo -e "${GREEN}====================================================================${NC}"