"""Integration tests for worker module."""

import pytest
from src.worker import Worker


class TestWorkerIntegration:
    """Integration tests for Worker class."""
    
    @pytest.mark.integration
    def test_worker_process_add(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "add", "a": 5, "b": 3})
        assert result == {"result": 8}
        assert worker.task_count == 1
    
    @pytest.mark.integration
    def test_worker_process_subtract(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "subtract", "a": 10, "b": 4})
        assert result == {"result": 6}
    
    @pytest.mark.integration
    def test_worker_process_multiply(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "multiply", "a": 3, "b": 7})
        assert result == {"result": 21}
    
    @pytest.mark.integration
    def test_worker_process_divide(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "divide", "a": 20, "b": 4})
        assert result == {"result": 5}
    
    @pytest.mark.integration
    def test_worker_process_divide_by_zero(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "divide", "a": 10, "b": 0})
        assert "error" in result
        assert "Division by zero" in result["error"]
    
    @pytest.mark.integration
    def test_worker_process_unknown_operation(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "unknown", "a": 1, "b": 2})
        assert "error" in result
        assert "Unknown operation" in result["error"]
    
    @pytest.mark.integration
    def test_worker_process_missing_operands(self):
        worker = Worker("test-worker")
        result = worker.process({"operation": "add"})
        assert result == {"error": "Missing operands"}
    
    @pytest.mark.integration
    def test_worker_multiple_tasks(self):
        worker = Worker("counter-worker")
        worker.process({"operation": "add", "a": 1, "b": 1})
        worker.process({"operation": "add", "a": 2, "b": 2})
        worker.process({"operation": "add", "a": 3, "b": 3})
        
        stats = worker.get_stats()
        assert stats["task_count"] == 3
        assert stats["name"] == "counter-worker"
    
    @pytest.mark.integration
    def test_worker_get_stats_initial(self):
        worker = Worker("stats-worker")
        stats = worker.get_stats()
        assert stats == {"name": "stats-worker", "task_count": 0}
