from Method.LearningForRingBySamplingGPU import LearningForRingBySampling
import numpy as np

def main():
    # 初始设定
    sample_n = 100 # 抽样的n
    n_lst = [10,20,30,40,50,60,70,80,90,100]
    pro_lst = list(np.arange(0,0.51,0.05)[1:])
    date_lst = range(1,61)

    # 特征设定
    features = ['Origin','Destination','O_Dig_0','D_Dig_0','O_Dig_1','D_Dig_1','O_Dig_2','D_Dig_2',  # 起点与终点特征
                'Distance_Dig_0','Distance_Dig_1','Distance_Dig_2',
                'Date','Actual In Time','Hour','Quarter',   # 进入时间特征
                'Acceleration','SlowingPro',  # 车辆特征
                'PathLength','Traffic Light Count', # 路径特征
                ] + [f'Traffic Light {i}' for i in range(10)] + [f'mean_TravelTime_before_{i}' for i in range(1,7)] + [f'std_TravelTime_before_{i}' for i in range(1,7)] # 时序特征
    target = ['Travel Time Log']

    # 初始化类
    lbs = LearningForRingBySampling(sample_n,
                                    n_lst,
                                    pro_lst,
                                    date_lst,
                                    features,
                                    target)

    # 学习设定
    sub_path = '1223'

    k = 10  # 只在设定有天数上限时才有用
    traindaynote = f'天数上限{k}'    # ['天数无上限','天数上限k']
    # traindaynote = '天数无上限'
    testnote = '测试为下一天'   # ['测试周期最后','测试周期抽取','测试为下一天']

    # 是否搜索参数
    is_search = False

    # 红绿灯变化的三个阶段日期
    days_set = [(1,20),(21,40),(41,60)]

    # 开始学习
    lbs.learning(sub_path,
                 traindaynote,
                 testnote,
                 is_search,
                 k,
                 days_set)

if __name__ == '__main__':
    main()