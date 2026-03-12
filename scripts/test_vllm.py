#!/usr/bin/env python3
"""
测试本地vllm服务的Python脚本
vllm默认提供OpenAI兼容的API，运行在8080端口
"""

import requests
import json
import time

# vllm服务地址
VLLM_BASE_URL = "http://localhost:8080/v1"


def calc_tokens_per_second(token_count: int, elapsed_seconds: float) -> float:
    """计算 token 吞吐率"""
    if token_count <= 0 or elapsed_seconds <= 0:
        return 0.0
    return token_count / elapsed_seconds


def print_efficiency_stats(label: str, elapsed_seconds: float, usage: dict | None):
    """打印响应耗时和 token/s"""
    usage = usage or {}
    prompt_tokens = usage.get("prompt_tokens", 0)
    completion_tokens = usage.get("completion_tokens", 0)
    total_tokens = usage.get("total_tokens", prompt_tokens + completion_tokens)
    completion_tps = calc_tokens_per_second(completion_tokens, elapsed_seconds)
    total_tps = calc_tokens_per_second(total_tokens, elapsed_seconds)

    print(f"\n{label}统计:")
    print(f"  耗时: {elapsed_seconds:.3f}s")
    print(f"  Prompt tokens: {prompt_tokens}")
    print(f"  Completion tokens: {completion_tokens}")
    print(f"  Total tokens: {total_tokens}")
    print(f"  Completion 吞吐: {completion_tps:.2f} tokens/s")
    print(f"  Overall 吞吐: {total_tps:.2f} tokens/s")

def list_models():
    """列出可用模型"""
    url = f"{VLLM_BASE_URL}/models"
    try:
        response = requests.get(url, timeout=10)
        if response.status_code == 200:
            models = response.json()
            print("可用模型:")
            for model in models.get("data", []):
                print(f"  - {model.get('id')}")
            return models
        else:
            print(f"获取模型列表失败: {response.status_code}")
            print(response.text)
            return None
    except Exception as e:
        print(f"连接失败: {e}")
        return None

def chat_completion(model: str, messages: list, temperature: float = 0.7, max_tokens: int = 256):
    """调用聊天完成接口"""
    url = f"{VLLM_BASE_URL}/chat/completions"
    
    payload = {
        "model": model,
        "messages": messages,
        "temperature": temperature,
        "max_tokens": max_tokens
    }
    
    try:
        start_time = time.perf_counter()
        response = requests.post(url, json=payload, timeout=240)
        elapsed_seconds = time.perf_counter() - start_time
        if response.status_code == 200:
            result = response.json()
            return result, elapsed_seconds
        else:
            print(f"请求失败: {response.status_code}")
            print(response.text)
            return None, elapsed_seconds
    except Exception as e:
        print(f"请求失败: {e}")
        return None, 0.0

def completion(prompt: str, model: str, max_tokens: int = 256):
    """调用基础补全接口"""
    url = f"{VLLM_BASE_URL}/completions"
    
    payload = {
        "model": model,
        "prompt": prompt,
        "max_tokens": max_tokens
    }
    
    try:
        start_time = time.perf_counter()
        response = requests.post(url, json=payload, timeout=60)
        elapsed_seconds = time.perf_counter() - start_time
        if response.status_code == 200:
            result = response.json()
            return result, elapsed_seconds
        else:
            print(f"请求失败: {response.status_code}")
            print(response.text)
            return None, elapsed_seconds
    except Exception as e:
        print(f"请求失败: {e}")
        return None, 0.0

def main():
    print("=" * 50)
    print("测试本地vllm服务 (localhost:8080)")
    print("=" * 50)
    
    # 1. 先列出可用模型
    print("\n[1] 获取模型列表...")
    models_data = list_models()
    
    if not models_data or not models_data.get("data"):
        print("无法获取模型列表，请确保vllm服务已启动")
        return
    
    # 使用第一个模型
    model_name = models_data["data"][0]["id"]
    print(f"\n使用模型: {model_name}")
    
    # 2. 测试聊天完成接口
    print("\n[2] 测试聊天完成接口...")
    messages = [
        {"role": "system", "content": "你是一个有帮助的AI助手。"},
        {"role": "user", "content": "请用一句话介绍你自己。"}
    ]
    
    result, elapsed_seconds = chat_completion(model_name, messages)
    if result:
        print("\n回复:")
        print(result["choices"][0]["message"]["content"])
        print(f"\nToken使用情况: {result.get('usage', {})}")
        print_efficiency_stats("聊天完成", elapsed_seconds, result.get("usage"))

    # 3. 测试流式聊天（可选）
    print("\n[3] 测试流式聊天...")
    url = f"{VLLM_BASE_URL}/chat/completions"
    payload = {
        "model": model_name,
        "messages": [{"role": "user", "content": "写一首关于春天的短诗"}],
        "stream": True,
        "stream_options": {"include_usage": True},
        "max_tokens": 200
    }
    
    try:
        start_time = time.perf_counter()
        first_token_latency = None
        usage = None
        response = requests.post(url, json=payload, stream=True, timeout=60)
        if response.status_code == 200:
            print("\n流式回复:")
            for line in response.iter_lines():
                if line:
                    line = line.decode('utf-8')
                    if line.startswith('data: '):
                        data = line[6:]
                        if data == '[DONE]':
                            break
                        try:
                            chunk = json.loads(data)
                            usage = chunk.get("usage") or usage
                            content = chunk.get("choices", [{}])[0].get("delta", {}).get("content", "")
                            if content and first_token_latency is None:
                                first_token_latency = time.perf_counter() - start_time
                            print(content, end="", flush=True)
                        except:
                            pass
            print("\n")
            elapsed_seconds = time.perf_counter() - start_time
            if first_token_latency is not None:
                print(f"首 token 延迟: {first_token_latency:.3f}s")
            print_efficiency_stats("流式聊天", elapsed_seconds, usage)
    except Exception as e:
        print(f"流式请求失败: {e}")

if __name__ == "__main__":
    main()
