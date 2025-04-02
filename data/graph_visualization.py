import json
import matplotlib.pyplot as plt
import networkx as nx
import numpy as np

# 读取JSON文件
def load_graph_from_json(file_path):
    with open(file_path, 'r', encoding='utf-8') as f:
        data = json.load(f)
    return data

# 创建图结构
def create_graph(data):
    G = nx.DiGraph()  # 创建有向图
    
    # 添加节点
    for node in data["nodes"]:
        G.add_node(node["id"], 
                  capacity=node["capacity"], 
                  maxSpeed=node["maxSpeed"], 
                  type=node["type"])
    
    # 添加边
    for edge in data["edges"]:
        G.add_edge(edge["from"], edge["to"])
    
    return G

# 绘制图结构
def visualize_graph(G, output_path=None):
    plt.figure(figsize=(20, 16))
    
    # 根据节点类型设置颜色
    node_colors = []
    for node, attrs in G.nodes(data=True):
        if attrs.get('type') == 'common':
            node_colors.append('skyblue')
        else:
            node_colors.append('salmon')
    
    # 使用Spring布局
    pos = nx.spring_layout(G, seed=42)
    
    # 绘制节点和边
    nx.draw_networkx_nodes(G, pos, node_size=50, node_color=node_colors, alpha=0.8)
    nx.draw_networkx_edges(G, pos, width=0.5, alpha=0.5, arrows=True, arrowsize=10)
    
    # 只为一部分节点显示标签，避免过度拥挤
    node_subset = list(G.nodes())[:min(100, len(G.nodes()))]
    nx.draw_networkx_labels(G, pos, {n: str(n) for n in node_subset}, font_size=8)
    
    plt.title("路网结构图")
    plt.axis('off')
    
    if output_path:
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
    
    plt.show()

# 为大型图创建更高级的可视化
def visualize_large_graph(G, output_path=None):
    plt.figure(figsize=(22, 18))
    
    # 根据节点类型和属性设置不同的颜色和大小
    node_colors = []
    node_sizes = []
    
    for node, attrs in G.nodes(data=True):
        # 根据节点类型设置颜色
        if attrs.get('type') == 'common':
            node_colors.append('skyblue')
        else:
            node_colors.append('salmon')
        
        # 根据容量设置大小
        node_sizes.append(30 + attrs.get('capacity', 1) * 10)
    
    # 使用Force Atlas 2布局算法，适合大型图
    try:
        pos = nx.kamada_kawai_layout(G)
    except:
        # 如果布局计算失败，则使用更简单的布局算法
        pos = nx.spring_layout(G, seed=42)
    
    # 绘制边
    nx.draw_networkx_edges(G, pos, width=0.3, alpha=0.3, arrows=True, arrowsize=5)
    
    # 绘制节点
    nx.draw_networkx_nodes(G, pos, node_size=node_sizes, node_color=node_colors, alpha=0.7)
    
    # 仅为一小部分重要节点添加标签
    # 选择度大的前N个节点
    degrees = dict(G.degree())
    top_nodes = sorted(degrees, key=degrees.get, reverse=True)[:50]
    nx.draw_networkx_labels(G, pos, {n: str(n) for n in top_nodes}, font_size=8)
    
    plt.title("路网结构图 (大型图优化视图)")
    plt.axis('off')
    
    if output_path:
        plt.savefig(output_path, dpi=300, bbox_inches='tight')
    
    plt.show()

# 计算并显示图的统计信息
def print_graph_stats(G):
    print(f"图统计信息:")
    print(f"节点数量: {G.number_of_nodes()}")
    print(f"边数量: {G.number_of_edges()}")
    
    # 计算连通分量
    if nx.is_weakly_connected(G):
        print("图是弱连通的")
    else:
        wcc = list(nx.weakly_connected_components(G))
        print(f"图有 {len(wcc)} 个弱连通分量")
        print(f"最大连通分量包含 {len(max(wcc, key=len))} 个节点")
    
    # 计算度的统计信息
    degrees = [d for n, d in G.degree()]
    print(f"平均度: {np.mean(degrees):.2f}")
    print(f"最大度: {max(degrees)}")
    
    # 节点类型统计
    node_types = {}
    for n, attr in G.nodes(data=True):
        node_type = attr.get('type', 'unknown')
        node_types[node_type] = node_types.get(node_type, 0) + 1
    
    print("节点类型分布:")
    for ntype, count in node_types.items():
        print(f"  - {ntype}: {count} ({count/G.number_of_nodes()*100:.1f}%)")

def main():
    # 文件路径
    file_path = 'data/2025032813200325_100_Graph.json'
    
    # 读取数据
    data = load_graph_from_json(file_path)
    
    # 创建图
    G = create_graph(data)
    
    # 打印图的统计信息
    print_graph_stats(G)
    
    # 绘制图像
    print("正在绘制基础图...")
    visualize_graph(G, "pics/road_network_basic.png")
    
    # 如果图很大，使用优化视图
    if G.number_of_nodes() > 500:
        print("检测到大型图，正在生成优化视图...")
        visualize_large_graph(G, "pics/road_network_optimized.png")
    
    print("图像绘制完成并已保存")

if __name__ == "__main__":
    main() 