import React, { useEffect, useState } from 'react';
import { getCategories, createCategory, deleteCategory, updateCategory } from '../../api/category';
import { Category } from '../../types';
import Pagination from '../../components/Pagination';
import InlineNotice from '../../components/InlineNotice';
import { getErrorMessage } from '../../utils/error';

const AdminCategories: React.FC = () => {
  const [categories, setCategories] = useState<Category[]>([]);
  const [name, setName] = useState('');
  const [slug, setSlug] = useState('');
  const [description, setDescription] = useState('');
  const [editingId, setEditingId] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [busyId, setBusyId] = useState<number | null>(null);
  const size = 10;

  const fetchList = async () => {
    try {
      const res = await getCategories({ page, size });
      setCategories(res.data.items || []);
      setTotal(res.data.total || 0);
    } catch (e) {
      setError(getErrorMessage(e, '分类列表加载失败'));
    }
  };

  useEffect(() => {
    fetchList();
  }, [page]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSubmitting(true);
    try {
      if (editingId) {
        await updateCategory(editingId, { name, slug, description });
      } else {
        await createCategory({ name, slug, description });
      }
      handleCancel();
      fetchList();
    } catch (e: any) {
      setError(getErrorMessage(e, '操作失败'));
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = (c: Category) => {
    setEditingId(c.id);
    setName(c.name);
    setSlug(c.slug);
    setDescription(c.description);
  };

  const handleCancel = () => {
    setEditingId(null);
    setName('');
    setSlug('');
    setDescription('');
  };

  const handleDelete = async (id: number) => {
    if (confirm('确认删除？')) {
      setError('');
      setBusyId(id);
      try {
        await deleteCategory(id);
        fetchList();
      } catch (e: any) {
        setError(getErrorMessage(e, '删除失败，可能存在关联文章'));
      } finally {
        setBusyId(null);
      }
    }
  };

  return (
    <div>
      <div className="mb-8 border-b border-mountain-grey pb-4">
        <h3 className="text-2xl font-bold text-ink tracking-widest">分类管理</h3>
      </div>

      <InlineNotice message={error} className="mb-6" />

      {/* Form */}
      <form onSubmit={handleSubmit} className="mb-8 flex gap-4 items-end">
        <div className="flex-1">
          <input type="text" placeholder="名称" required value={name} onChange={e => setName(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-none focus:border-ochre text-ink" />
        </div>
        <div className="flex-1">
          <input type="text" placeholder="Slug" required value={slug} onChange={e => setSlug(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-none focus:border-ochre text-ink" />
        </div>
        <div className="flex-1">
          <input type="text" placeholder="描述" value={description} onChange={e => setDescription(e.target.value)} className="w-full bg-transparent border-b border-mountain-grey py-2 focus:outline-none focus:border-ochre text-ink" />
        </div>
        <div className="flex space-x-2">
          <button type="submit" disabled={submitting} className="px-4 py-2 bg-ink text-paper tracking-widest text-sm hover:bg-opacity-80 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
            {submitting ? '处理中...' : editingId ? '保存' : '新增'}
          </button>
          {editingId && (
            <button type="button" onClick={handleCancel} className="px-4 py-2 border border-mountain-grey text-ink tracking-widest text-sm hover:border-ink transition-colors">
              取消
            </button>
          )}
        </div>
      </form>

      <table className="w-full text-left border-collapse text-sm">
        <thead>
          <tr className="border-b border-mountain-grey text-ink-light opacity-80 tracking-widest">
            <th className="py-3 font-normal">名称</th>
            <th className="py-3 font-normal">Slug</th>
            <th className="py-3 font-normal text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {categories.map(c => (
            <tr key={c.id} className="border-b border-mountain-grey border-opacity-50 hover:bg-mountain-grey hover:bg-opacity-20 transition-colors text-ink">
              <td className="py-4 font-bold">{c.name}</td>
              <td className="py-4 text-ink-light">{c.slug}</td>
              <td className="py-4 text-right space-x-4">
                <button onClick={() => handleEdit(c)} className="text-ink opacity-80 hover:text-ochre hover:opacity-100 tracking-wider">修编</button>
                <button onClick={() => handleDelete(c.id)} disabled={busyId === c.id} className="text-ochre opacity-80 hover:opacity-100 tracking-wider disabled:opacity-50 disabled:cursor-not-allowed">销毁</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination 
        currentPage={page} 
        total={total} 
        pageSize={size} 
        onPageChange={setPage} 
      />
    </div>
  );
};

export default AdminCategories;
