"""Worker module for processing tasks."""

from typing import Any
from .calculator import add, subtract, multiply, divide


class Worker:
    """Worker class for processing calculation tasks."""
    
    def __init__(self, name: str = "default"):
        self.name = name
        self.task_count = 0
    
    def process(self, task: dict[str, Any]) -> dict[str, Any]:
        """Process a calculation task.
        
        Args:
            task: Dictionary with 'operation', 'a', and 'b' keys.
            
        Returns:
            Dictionary with 'result' or 'error' key.
        """
        self.task_count += 1
        
        operation = task.get("operation")
        a = task.get("a")
        b = task.get("b")
        
        if a is None or b is None:
            return {"error": "Missing operands"}
        
        try:
            if operation == "add":
                result = add(a, b)
            elif operation == "subtract":
                result = subtract(a, b)
            elif operation == "multiply":
                result = multiply(a, b)
            elif operation == "divide":
                result = divide(a, b)
            else:
                return {"error": f"Unknown operation: {operation}"}
            
            return {"result": result}
        except Exception as e:
            return {"error": str(e)}
    
    def get_stats(self) -> dict[str, Any]:
        """Get worker statistics."""
        return {
            "name": self.name,
            "task_count": self.task_count,
        }
