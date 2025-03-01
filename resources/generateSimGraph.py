import pandas as pd

file_path = 'resources/Barcelona_net.csv'
df = pd.read_csv(file_path)

df['init_node'] = df['init_node'].astype(int)
df['term_node'] = df['term_node'].astype(int)

def generate_new_edges_and_nodes(init_node, term_node, length, speed):
    new_node = f"{int(init_node)}000{int(term_node)}"
    new_edges = [
        [init_node, new_node],
        [new_node, term_node]
    ]
    new_node_info = [new_node, length, speed]
    return new_edges, new_node_info

new_edges_list = []
new_nodes_list = []
for _, row in df.iterrows():
    new_edges, new_node_info = generate_new_edges_and_nodes(
        row['init_node'], row['term_node'], row['length'], row['speed']
    )
    new_edges_list.extend(new_edges)
    new_nodes_list.append(new_node_info)

original_nodes = set(df['init_node']).union(set(df['term_node']))
for node in original_nodes:
    # speed = df.loc[df['init_node'] == node, 'speed'].values[0]
    
    # Barcelona
    speed = 5
    new_nodes_list.append([node, -1, speed])

df_edges = pd.DataFrame(new_edges_list, columns=['init_node', 'term_node'])
df_edges['init_node'] = df_edges['init_node'].astype(int).astype(str)
df_edges['term_node'] = df_edges['term_node'].astype(int).astype(str)

df_nodes = pd.DataFrame(new_nodes_list, columns=['node_id', 'length', 'speed'])
df_nodes['node_id'] = df_nodes['node_id'].astype(str)

# df_nodes['length'] = df_nodes['length'].apply(lambda x: round(x / 7.5) if x != -1 else -1)
# df_nodes['speed'] = df_nodes['speed'].apply(lambda x: round(x / 5))
df_nodes['length'] = df_nodes['length'].apply(lambda x: round(x * 1000 * 1.609 / 7.5) if x != -1 else -1)

df_edges.to_csv('./resources/Barcelona_Edges.csv', index=False)
df_nodes.to_csv('./resources/Barcelona_Nodes.csv', index=False)